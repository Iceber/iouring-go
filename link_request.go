// +build linux

package iouring

import (
	"errors"
	"time"
	"unsafe"

	iouring_syscall "github.com/iceber/iouring-go/syscall"
	"golang.org/x/sys/unix"
)

func (iour *IOURing) SubmitLinkRequests(requests []Request, ch chan<- *Result) error {
	return iour.submitLinkRequest(requests, ch, false)
}

func (iour *IOURing) SubmitHardLinkRequests(requests []Request, ch chan<- *Result) error {
	return iour.submitLinkRequest(requests, ch, true)
}

func (iour *IOURing) submitLinkRequest(requests []Request, ch chan<- *Result, hard bool) error {
	// TODO(iceber): no length limit
	if len(requests) > int(*iour.sq.entries) {
		return errors.New("requests is too many")
	}

	flags := iouring_syscall.IOSQE_FLAGS_IO_LINK
	if hard {
		flags = iouring_syscall.IOSQE_FLAGS_IO_HARDLINK
	}

	iour.submitLock.Lock()
	defer iour.submitLock.Unlock()

	var sqeN uint32
	for i := range requests {
		sqe := iour.getSQEntry()
		sqeN++

		if _, err := iour.doRequest(sqe, requests[i], ch); err != nil {
			iour.sq.fallback(sqeN)
			return err
		}

		sqe.CleanFlags(iouring_syscall.IOSQE_FLAGS_IO_HARDLINK | iouring_syscall.IOSQE_FLAGS_IO_LINK)
		if i < len(requests)-1 {
			sqe.SetFlags(flags)
		}
	}

	_, err := iour.submit()
	return err
}

func linkTimeout(t time.Duration) Request {
	timespec := unix.NsecToTimespec(t.Nanoseconds())

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.hold(&timespec)
		userData.result.resolver = timeoutResolver

		sqe.PrepOperation(iouring_syscall.IORING_OP_LINK_TIMEOUT, -1, uint64(uintptr(unsafe.Pointer(&timespec))), 1, 0)
	}
}
