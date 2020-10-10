// +build linux

package iouring

import (
	"time"

	iouring_syscall "github.com/iceber/iouring-go/syscall"
)

type IOURingOption func(*IOURing)

func WithSQPoll() IOURingOption {
	return func(iour *IOURing) {
		iour.params.Flags |= iouring_syscall.IORING_SETUP_FLAGS_SQPOLL
	}
}

func WithSQPollThreadCPU(cpu uint32) IOURingOption {
	return func(iour *IOURing) {
		iour.params.Flags |= iouring_syscall.IORING_SETUP_FLAGS_SQ_AFF
		iour.params.SQThreadCPU = cpu
	}
}

func WithSQPollThreadIdle(idle time.Duration) IOURingOption {
	return func(iour *IOURing) {
		iour.params.SQThreadIdle = uint32(idle / time.Millisecond)
	}
}

func WithParams(params *iouring_syscall.IOURingParams) IOURingOption {
	return func(iour *IOURing) {
		iour.params = params
	}
}

func WithCQSize(size uint32) IOURingOption {
	return func(iour *IOURing) {
		iour.params.Flags |= iouring_syscall.IORING_SETUP_FLAGS_CQSIZE
		iour.params.CQEntries = size
	}
}

func WithAsync() IOURingOption {
	return func(iour *IOURing) {
		iour.async = true
	}
}
