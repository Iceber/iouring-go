// +build linux

package iouring

import (
	"errors"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	iouring_syscall "github.com/iceber/iouring-go/syscall"
)

func (iour *IOURing) SubmitLinkRequests(requests []Request, ch chan<- *Result) (*ResultGroup, error) {
	return iour.submitLinkRequest(requests, ch, false)
}

func (iour *IOURing) SubmitHardLinkRequests(requests []Request, ch chan<- *Result) (*ResultGroup, error) {
	return iour.submitLinkRequest(requests, ch, true)
}

func (iour *IOURing) submitLinkRequest(requests []Request, ch chan<- *Result, hard bool) (*ResultGroup, error) {
	// TODO(iceber): no length limit
	if len(requests) > int(*iour.sq.entries) {
		return nil, errors.New("too many requests")
	}

	flags := iouring_syscall.IOSQE_FLAGS_IO_LINK
	if hard {
		flags = iouring_syscall.IOSQE_FLAGS_IO_HARDLINK
	}

	iour.submitLock.Lock()
	defer iour.submitLock.Unlock()

	if iour.IsClosed() {
		return nil, ErrIOURingClosed
	}

	var sqeN uint32
	userDatas := make([]*UserData, 0, len(requests))
	for i := range requests {
		sqe := iour.getSQEntry()
		sqeN++

		userData, err := iour.doRequest(sqe, requests[i], ch)
		if err != nil {
			iour.sq.fallback(sqeN)
			return nil, err
		}
		userDatas = append(userDatas, userData)

		sqe.CleanFlags(iouring_syscall.IOSQE_FLAGS_IO_HARDLINK | iouring_syscall.IOSQE_FLAGS_IO_LINK)
		if i < len(requests)-1 {
			sqe.SetFlags(flags)
		}
	}

	iour.userDataLock.Lock()
	for _, data := range userDatas {
		iour.userDatas[data.id] = data
	}
	iour.userDataLock.Unlock()

	if _, err := iour.submit(); err != nil {
		iour.userDataLock.Lock()
		for _, data := range userDatas {
			delete(iour.userDatas, data.id)
		}
		iour.userDataLock.Unlock()

		return nil, err
	}

	return newResultGroup(userDatas), nil
}

func linkTimeout(t time.Duration) Request {
	timespec := unix.NsecToTimespec(t.Nanoseconds())

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.hold(&timespec)
		userData.result.resolver = timeoutResolver

		sqe.PrepOperation(iouring_syscall.IORING_OP_LINK_TIMEOUT, -1, uint64(uintptr(unsafe.Pointer(&timespec))), 1, 0)
	}
}
