// +build linux

package iouring

import iouring_syscall "github.com/iceber/iouring-go/syscall"

func (iour *IOURing) SubmitLinkedRequest(requests []IORequest, ch chan<- *Result) error {
	return iour.submitLinkedRequest(requests, ch, false)
}

func (iour *IOURing) SubmitHardlinkedRequest(requests []IORequest, ch chan<- *Result) error {
	return iour.submitLinkedRequest(requests, ch, true)
}

func (iour *IOURing) submitLinkedRequest(requests []IORequest, ch chan<- *Result, hard bool) error {
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

		if i < len(requests)-1 {
			sqe.SetFlags(flags)
		}
	}

	_, err := iour.submit()
	return err
}
