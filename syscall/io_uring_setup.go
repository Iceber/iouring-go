// +build linux

package iouring_syscall

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	IORING_SETUP_FLAGS_IOPOLL uint32 = 1 << iota
	IORING_SETUP_FLAGS_SQPOLL
	IORING_SETUP_FLAGS_SQ_AFF
	IORING_SETUP_FLAGS_CQSIZE
	IORING_SETUP_FLAGS_CLAMP
	IORING_SETUP_FLAGS_ATTACH_WQ
	IORING_SETUP_FLAGS_R_DISABLED
)

const (
	IORING_FEAT_SINGLE_MMAP uint32 = 1 << iota
	IORING_FEAT_NODROP
	IORING_FEAT_SUBMIT_STABLE
	IORING_FEAT_RW_CUR_POS
	IORING_FEAT_CUR_PERSONALITY
	IORING_FEAT_FAST_POLL
	IORING_FEAT_POLL_32BITS
	IORING_FEAT_SQPOLL_NONFIXED
)

type IOURingParams struct {
	SQEntries    uint32
	CQEntries    uint32
	Flags        uint32
	SQThreadCPU  uint32
	SQThreadIdle uint32
	Features     uint32
	WQFd         uint32
	Resv         [3]uint32

	SQOffset SubmissionQueueRingOffset
	CQOffset CompletionQueueRingOffset
}

type SubmissionQueueRingOffset struct {
	Head        uint32
	Tail        uint32
	RingMask    uint32
	RingEntries uint32
	Flags       uint32
	Dropped     uint32
	Array       uint32
	Resv1       uint32
	Resv2       uint32
}

type CompletionQueueRingOffset struct {
	Head     uint32
	Tail     uint32
	RingMask uint32
	Entries  uint32
	Overflow uint32
	Cqes     uint32
	Flags    uint32
	Resv     [2]uint64
}

func IOURingSetup(entries uint, params *IOURingParams) (int, error) {
	res, _, errno := syscall.RawSyscall(
		SYS_IO_URING_SETUP,
		uintptr(entries),
		uintptr(unsafe.Pointer(params)),
		0,
	)
	if errno != 0 {
		return int(res), os.NewSyscallError("iouring_setup", errno)
	}

	return int(res), nil
}
