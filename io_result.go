// +build linux

package iouring

import (
	"sync"
	"syscall"

	"github.com/pkg/errors"

	iouring_syscall "github.com/iceber/iouring-go/syscall"
)

type ResultResolver func(result *Result)

type Result struct {
	opcode uint8
	res    int32

	once     sync.Once
	resolver ResultResolver

	b0 *[]byte
	b1 *[]byte
	bs *[][]byte

	err error
	r0  interface{}
	r1  interface{}

	requestInfo interface{}
}

func (result *Result) load(cqe *iouring_syscall.CompletionQueueEvent) {
	result.res = cqe.Result
}

func (result *Result) resolve() {
	result.once.Do(func() {
		if result.resolver == nil {
			return
		}

		result.resolver(result)
		result.resolver = nil
	})
}

func (result *Result) Opcode() uint8 {
	return result.opcode
}

func (result *Result) GetRequestBuffer() (b0, b1 *[]byte) {
	return result.b0, result.b1
}

func (result *Result) GetRequestBuffers() *[][]byte {
	return result.bs
}

func (result *Result) GetRequestInfo() interface{} {
	return result.requestInfo
}

func (result *Result) ReturnValue0() interface{} {
	result.resolve()
	return result.r0
}

func (result *Result) ReturnValue1() interface{} {
	result.resolve()
	return result.r1
}

func (result *Result) Err() error {
	result.resolve()
	return result.err
}

func (result *Result) ReturnFd() (int, error) {
	return result.ReturnInt()
}

func (result *Result) ReturnInt() (int, error) {
	result.resolve()

	if result.err != nil {
		return -1, result.err
	}

	fd, ok := result.r0.(int)
	if !ok {
		return -1, errors.New("")
	}

	return fd, nil
}

func errResolver(result *Result) {
	if result.res < 0 {
		result.err = syscall.Errno(-result.res)
		if result.err == syscall.ECANCELED {
			result.err = IOURING_ERROR_CANCELED
		}
	}
}

func fdResolver(result *Result) {
	if errResolver(result); result.err != nil {
		return
	}
	result.r0 = int(result.res)
}

func timeoutResolver(result *Result) {
	if errResolver(result); result.err != nil {
		// timeout completion
		if result.err == syscall.ETIME {
			result.err = nil
			result.r0 = IOURING_TIMEOUT
		}
		return
	}

	if result.res == 0 {
		result.r0 = IOURING_TIMEOUT_WITH_CQE_COUNT
	}
}

func removeTimeoutResolver(result *Result) {
	if errResolver(result); result.err != nil {
		switch result.err {
		case syscall.EBUSY:
			result.err = errors.Wrap(result.err, "already timeout")
		case syscall.ENOENT:
			result.err = errors.Wrap(result.err, "timeout request not found")
		}
	}
	if result.res == 0 {
		result.r0 = IOURING_TIMEOUT_WITH_CQE_COUNT
		return
	}
}

func cancelResolver(result *Result) {
	if errResolver(result); result.err != nil {
		switch result.err {
		case syscall.ENOENT:
			result.err = errors.Wrap(result.err, "request not found")
		case syscall.EALREADY:
			result.err = nil
			result.r0 = IOURING_REQUEST_MAYBE_CANCELED
		}
	}

	if result.res == 0 {
		result.r0 = IOURING_FOUND_REQUEST
	}
}
