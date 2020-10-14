// +build linux

package iouring

import (
	"errors"
	"sync"
	"syscall"
)

type ResultResolver func(result *Result)

type Result struct {
	id     uint64
	opcode uint8
	res    int32

	once     sync.Once
	resolver ResultResolver

	fd int
	b0 *[]byte
	b1 *[]byte
	bs *[][]byte

	err error
	r0  interface{}
	r1  interface{}

	requestInfo interface{}
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

func (result *Result) ID() uint64 {
	return result.id
}

func (result *Result) Opcode() uint8 {
	return result.opcode
}

func (result *Result) Fd() int {
	return result.fd
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

func (result *Result) Err() error {
	result.resolve()
	return result.err
}

func (result *Result) ReturnValue0() interface{} {
	result.resolve()
	return result.r0
}

func (result *Result) ReturnValue1() interface{} {
	result.resolve()
	return result.r1
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
		return -1, errors.New("result value is not int")
	}

	return fd, nil
}

func errResolver(result *Result) {
	if result.res < 0 {
		result.err = syscall.Errno(-result.res)
		if result.err == syscall.ECANCELED {
			// request is canceled
			result.err = ErrRequestCanceled
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
		// if timeout got completed through expiration of the timer
		// result.res is -ETIME and result.err is syscall.ETIME
		if result.err == syscall.ETIME {
			result.err = nil
			result.r0 = TimeoutExpiration
		}
		return
	}

	// if timeout got completed through requests completing
	// result.res is 0
	if result.res == 0 {
		result.r0 = CountCompletion
	}
}

func removeTimeoutResolver(result *Result) {
	if errResolver(result); result.err != nil {
		switch result.err {
		case syscall.EBUSY:
			// timeout request was found bu expiration was already in progress
			result.err = ErrRequestCompleted
		case syscall.ENOENT:
			// timeout request not found
			result.err = ErrRequestNotFound
		}
		return
	}

	// timeout request is found and cacelled successfully
	// result.res value is 0
}

func cancelResolver(result *Result) {
	if errResolver(result); result.err != nil {
		switch result.err {
		case syscall.ENOENT:
			result.err = ErrRequestNotFound
		case syscall.EALREADY:
			result.err = nil
			result.r0 = RequestMaybeCanceled
		}
		return
	}

	if result.res == 0 {
		result.r0 = RequestCanceledSuccessfully
	}
}
