// +build linux

package iouring

import (
	"errors"
	"sync"
	"sync/atomic"
	"syscall"

	iouring_syscall "github.com/iceber/iouring-go/syscall"
)

type ResultResolver func(result *Result)

type Result struct {
	iour *IOURing

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

	group *ResultGroup
	done  chan struct{}
}

func (result *Result) resolve() {
	if result.resolver == nil {
		return
	}

	select {
	case <-result.done:
	default:
		return
	}

	result.once.Do(func() {
		result.resolver(result)
		result.resolver = nil
	})
}

func (result *Result) complate(cqe *iouring_syscall.CompletionQueueEvent) {
	result.res = cqe.Result
	result.iour = nil
	close(result.done)

	if result.group != nil {
		result.group.complateOne()
		result.group = nil
	}
}

// Cancel request if request is not completed
func (result *Result) Cancel() (*Result, error) {
	select {
	case <-result.done:
		return nil, ErrRequestCompleted
	default:
	}

	return result.iour.submitCancel(result.id)
}

func (result *Result) Done() <-chan struct{} {
	return result.done
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

func (result *Result) FreeRequestBuffer() {
	result.b0 = nil
	result.b1 = nil
	result.bs = nil
}

type ResultGroup struct {
	results []*Result

	complates int32
	done      chan struct{}
}

func newResultGroup(userData []*UserData) *ResultGroup {
	group := &ResultGroup{
		results: make([]*Result, len(userData)),
		done:    make(chan struct{}),
	}

	for i, data := range userData {
		group.results[i] = data.result
		data.result.group = group
	}
	return group
}

func (group *ResultGroup) complateOne() {
	if atomic.AddInt32(&group.complates, 1) == int32(len(group.results)) {
		close(group.done)
	}
}

func (group *ResultGroup) Len() int {
	return len(group.results)
}

func (group *ResultGroup) Done() <-chan struct{} {
	return group.done
}

func (group *ResultGroup) Results() []*Result {
	return group.results
}

func (group *ResultGroup) ErrResults() (results []*Result) {
	for _, result := range group.results {
		if result.Err() != nil {
			results = append(results, result)
		}
	}
	return
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
