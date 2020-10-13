// +build linux

package iouring

import (
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	iouring_syscall "github.com/iceber/iouring-go/syscall"
)

const (
	IOURING_TIMEOUT                = 0
	IOURING_TIMEOUT_WITH_CQE_COUNT = 1
)

func RequestWithTimeout(request IORequest, timeout time.Duration) []IORequest {
	linkRequest := func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		request(sqe, userData)
		sqe.SetFlags(iouring_syscall.IOSQE_FLAGS_IO_LINK)
	}
	return []IORequest{linkRequest, linkTimeout(timeout)}
}

func Timeout(t time.Duration) IORequest {
	timespec := unix.NsecToTimespec(t.Nanoseconds())

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.hold(&timespec)
		userData.result.resolver = timeoutResolver

		sqe.PrepOperation(iouring_syscall.IORING_OP_TIMEOUT, -1, uint64(uintptr(unsafe.Pointer(&timespec))), 1, 0)
	}
}

func TimeoutWithTime(t time.Time) (IORequest, error) {
	timespec, err := unix.TimeToTimespec(t)
	if err != nil {
		return nil, err
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.hold(&timespec)
		userData.result.resolver = timeoutResolver

		sqe.PrepOperation(iouring_syscall.IORING_OP_TIMEOUT, -1, uint64(uintptr(unsafe.Pointer(&timespec))), 1, 0)
		sqe.SetOpFlags(iouring_syscall.IORING_TIMEOUT_ABS)
	}, nil
}

func CountCompletionEvent(n uint64) IORequest {
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = timeoutResolver

		sqe.PrepOperation(iouring_syscall.IORING_OP_TIMEOUT, -1, 0, 0, n)
	}
}

func RemoveTimeout(id uint64) IORequest {
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = removeTimeoutResolver

		sqe.PrepOperation(iouring_syscall.IORING_OP_TIMEOUT, -1, id, 0, 0)
	}
}
