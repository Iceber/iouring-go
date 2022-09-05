// +build linux

package iouring

import (
	"fmt"
	"os"
	"reflect"
	"syscall"
	"unsafe"

	iouring_syscall "github.com/iceber/iouring-go/syscall"
)

var (
	uint32Size = uint32(unsafe.Sizeof(uint32(0)))
	sqeSize    = uint32(unsafe.Sizeof(iouring_syscall.SubmissionQueueEntry{}))
	cqeSize    = uint32(unsafe.Sizeof(iouring_syscall.CompletionQueueEvent{}))
)

func mmapIOURing(iour *IOURing) (err error) {
	defer func() {
		if err != nil {
			munmapIOURing(iour)
		}
	}()
	iour.sq = new(SubmissionQueue)
	iour.cq = new(CompletionQueue)

	iour.sq.size = iour.params.SQOffset.Array + iour.params.SQEntries*uint32Size
	iour.cq.size = iour.params.CQOffset.Cqes + iour.params.CQEntries*cqeSize
	if (iour.params.Features & iouring_syscall.IORING_FEAT_SINGLE_MMAP) != 0 {
		if iour.cq.size > iour.sq.size {
			iour.sq.size = iour.cq.size
		} else {
			iour.cq.size = iour.sq.size
		}
	}

	if err = mmapSQ(iour.fd, iour.params, iour.sq); err != nil {
		return err
	}

	if (iour.params.Features & iouring_syscall.IORING_FEAT_SINGLE_MMAP) != 0 {
		iour.cq.ptr = iour.sq.ptr
	}

	if err = mmapCQ(iour.fd, iour.params, iour.cq); err != nil {
		return err
	}

	if err = mmapSQEs(iour.fd, iour.params, iour.sq); err != nil {
		return err
	}
	return nil
}

func mmapSQ(fd int, params *iouring_syscall.IOURingParams, sq *SubmissionQueue) (err error) {
	sq.ptr, err = mmap(fd, sq.size, iouring_syscall.IORING_OFF_SQ_RING)
	if err != nil {
		return fmt.Errorf("mmap sq ring: %w", err)
	}

	sq.head = (*uint32)(unsafe.Pointer(sq.ptr + uintptr(params.SQOffset.Head)))
	sq.tail = (*uint32)(unsafe.Pointer(sq.ptr + uintptr(params.SQOffset.Tail)))
	sq.mask = (*uint32)(unsafe.Pointer(sq.ptr + uintptr(params.SQOffset.RingMask)))
	sq.entries = (*uint32)(unsafe.Pointer(sq.ptr + uintptr(params.SQOffset.RingEntries)))
	sq.flags = (*uint32)(unsafe.Pointer(sq.ptr + uintptr(params.SQOffset.Flags)))
	sq.dropped = (*uint32)(unsafe.Pointer(sq.ptr + uintptr(params.SQOffset.Dropped)))

	sq.array = *(*[]uint32)(unsafe.Pointer(&reflect.SliceHeader{
		Data: sq.ptr + uintptr(params.SQOffset.Array),
		Len:  int(params.SQEntries),
		Cap:  int(params.SQEntries),
	}))
	return nil
}

func mmapCQ(fd int, params *iouring_syscall.IOURingParams, cq *CompletionQueue) (err error) {
	if cq.ptr == 0 {
		cq.ptr, err = mmap(fd, cq.size, iouring_syscall.IORING_OFF_CQ_RING)
		if err != nil {
			return fmt.Errorf("mmap cq ring: %w", err)
		}
	}

	cq.head = (*uint32)(unsafe.Pointer(cq.ptr + uintptr(params.CQOffset.Head)))
	cq.tail = (*uint32)(unsafe.Pointer(cq.ptr + uintptr(params.CQOffset.Tail)))
	cq.mask = (*uint32)(unsafe.Pointer(cq.ptr + uintptr(params.CQOffset.RingMask)))
	cq.entries = (*uint32)(unsafe.Pointer(cq.ptr + uintptr(params.CQOffset.RingEntries)))
	cq.flags = (*uint32)(unsafe.Pointer(cq.ptr + uintptr(params.CQOffset.Flags)))
	cq.overflow = (*uint32)(unsafe.Pointer(cq.ptr + uintptr(params.CQOffset.Overflow)))

	cq.cqes = *(*[]iouring_syscall.CompletionQueueEvent)(
		unsafe.Pointer(&reflect.SliceHeader{
			Data: cq.ptr + uintptr(params.CQOffset.Cqes),
			Len:  int(params.CQEntries),
			Cap:  int(params.CQEntries),
		}))
	return nil
}

func mmapSQEs(fd int, params *iouring_syscall.IOURingParams, sq *SubmissionQueue) (err error) {
	ptr, err := mmap(fd, params.SQEntries*sqeSize, iouring_syscall.IORING_OFF_SQES)
	if err != nil {
		return fmt.Errorf("mmap sqe array: %w", err)
	}

	sq.sqes = *(*[]iouring_syscall.SubmissionQueueEntry)(
		unsafe.Pointer(&reflect.SliceHeader{
			Data: ptr,
			Len:  int(params.SQEntries),
			Cap:  int(params.SQEntries),
		}))
	return nil
}

func munmapIOURing(iour *IOURing) error {
	if iour.sq != nil && iour.sq.ptr != 0 {
		if len(iour.sq.sqes) != 0 {
			err := munmap(uintptr(unsafe.Pointer(&iour.sq.sqes[0])), uint32(len(iour.sq.sqes))*sqeSize)
			if err != nil {
				return fmt.Errorf("ummap sqe array: %w", err)
			}
			iour.sq.sqes = nil
		}

		if err := munmap(iour.sq.ptr, iour.sq.size); err != nil {
			return fmt.Errorf("munmap sq: %w", err)
		}
		if iour.sq.ptr == iour.cq.ptr {
			iour.cq = nil
		}
		iour.sq = nil
	}

	if iour.cq != nil && iour.cq.ptr != 0 {
		if err := munmap(iour.cq.ptr, iour.cq.size); err != nil {
			return fmt.Errorf("munmap cq: %w", err)
		}
		iour.cq = nil
	}
	return nil
}

func mmap(fd int, length uint32, offset uint64) (uintptr, error) {
	ptr, _, errno := syscall.Syscall6(
		syscall.SYS_MMAP,
		0,
		uintptr(length),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED|syscall.MAP_POPULATE,
		uintptr(fd),
		uintptr(offset),
	)
	if errno != 0 {
		return 0, os.NewSyscallError("mmap", errno)
	}
	return uintptr(ptr), nil
}

func munmap(ptr uintptr, length uint32) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_MUNMAP,
		ptr,
		uintptr(length),
		0,
	)
	if errno != 0 {
		return os.NewSyscallError("munmap", errno)
	}
	return nil
}
