// +build linux

package iouring

import (
	"errors"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	iouring_syscall "github.com/iceber/iouring-go/syscall"
)

// IOURing contains iouring_syscall submission and completion queue.
// It's safe for concurrent use by multiple goroutines.
type IOURing struct {
	params *iouring_syscall.IOURingParams
	fd     int

	sq *SubmissionQueue
	cq *CompletionQueue

	async bool
	Flags uint32

	submitLock sync.Mutex
	submits    int64
	submitSign chan struct{}

	userDataLock sync.RWMutex
	userDatas    map[uint64]*UserData

	fileRegister FileRegister
}

// New return a IOURing instance by IOURingOptions
func New(entries uint, opts ...IOURingOption) (iour *IOURing, err error) {
	iour = &IOURing{
		params:     &iouring_syscall.IOURingParams{},
		userDatas:  make(map[uint64]*UserData),
		submitSign: make(chan struct{}, 1),
	}

	for _, opt := range opts {
		opt(iour)
	}

	iour.fd, err = iouring_syscall.IOURingSetup(entries, iour.params)
	if err != nil {
		log.Println("setup", err)
		return nil, err
	}

	if err := mmapIOURing(iour); err != nil {
		log.Println("mmap", err)
		return nil, err
	}

	iour.fileRegister = &fileRegister{
		iouringFd:    iour.fd,
		sparseIndexs: make(map[int]int),
	}
	iour.Flags = iour.params.Flags

	go iour.run()
	return iour, nil
}

// TODO(iceber): get available entry use async notification
func (iour *IOURing) getSQEntry() *iouring_syscall.SubmissionQueueEntry {
	for {
		sqe := iour.sq.GetSQEntry()
		if sqe != nil {
			return sqe
		}
		runtime.Gosched()
	}
}

func (iour *IOURing) doRequest(sqe *iouring_syscall.SubmissionQueueEntry, request Request, ch chan<- *Result) (id uint64, err error) {
	// TODO(iceber): use sync.Poll
	userData := makeUserData(ch)

	request(sqe, userData)
	userData.setOpcode(sqe.Opcode())

	id = uint64(uintptr(unsafe.Pointer(userData)))
	iour.userDataLock.Lock()
	iour.userDatas[id] = userData
	iour.userDataLock.Unlock()
	sqe.SetUserData(id)

	if sqe.Fd() >= 0 {
		if index, ok := iour.fileRegister.GetFileIndex(int32(sqe.Fd())); ok {
			sqe.SetFdIndex(int32(index))
		} else if iour.Flags&iouring_syscall.IORING_SETUP_FLAGS_SQPOLL != 0 {
			return 0, errors.New("fd is not registered")
		}
	}

	if iour.async {
		sqe.SetFlags(iouring_syscall.IOSQE_FLAGS_ASYNC)
	}
	return
}

// SubmitRequest by Request function and io result is notified via channel
// return request id, can be used to cancel a request
func (iour *IOURing) SubmitRequest(request Request, ch chan<- *Result) (uint64, error) {
	iour.submitLock.Lock()
	defer iour.submitLock.Unlock()

	sqe := iour.getSQEntry()
	id, err := iour.doRequest(sqe, request, ch)
	if err != nil {
		iour.sq.fallback(1)
		return id, err
	}

	_, err = iour.submit()
	return id, err
}

// SubmitRequests by Request functions and io results are notified via channel
func (iour *IOURing) SubmitRequests(requests []Request, ch chan<- *Result) error {
	// TODO(iceber): no length limit
	if len(requests) > int(*iour.sq.entries) {
		return errors.New("requests is too many")
	}

	iour.submitLock.Lock()
	defer iour.submitLock.Unlock()

	var sqeN uint32
	for _, request := range requests {
		sqe := iour.getSQEntry()
		sqeN++

		if _, err := iour.doRequest(sqe, request, ch); err != nil {
			iour.sq.fallback(sqeN)
			return err
		}
	}
	_, err := iour.submit()
	return err
}

func (iour *IOURing) needEnter(flags *uint32) bool {
	if (iour.Flags & iouring_syscall.IORING_SETUP_FLAGS_SQPOLL) == 0 {
		return true
	}

	if iour.sq.needWakeup() {
		*flags |= iouring_syscall.IORING_SQ_NEED_WAKEUP
		return true
	}
	return false
}

func (iour *IOURing) submit() (submitted int, err error) {
	submitted = iour.sq.flush()
	defer func() {
		if err != nil {
			return
		}
		atomic.AddInt64(&iour.submits, int64(submitted))

		select {
		case iour.submitSign <- struct{}{}:
		default:
		}
	}()

	var flags uint32
	if !iour.needEnter(&flags) || submitted == 0 {
		return
	}

	if (iour.Flags & iouring_syscall.IORING_SETUP_FLAGS_IOPOLL) != 0 {
		flags |= iouring_syscall.IORING_ENTER_FLAGS_GETEVENTS
	}

	submitted, err = iouring_syscall.IOURingEnter(iour.fd, uint32(submitted), 0, flags, nil)
	return
}

/*
func (iour *IOURing) submitAndWait(waitCount uint32) (submitted int, err error) {
	submitted = iour.sq.flush()

	var flags uint32
	if !iour.needEnter(&flags) && waitCount == 0 {
		return
	}

	if waitCount != 0 || (iour.Flags&iouring_syscall.IORING_SETUP_FLAGS_IOPOLL) != 0 {
		flags |= iouring_syscall.IORING_ENTER_FLAGS_GETEVENTS
	}

	submitted, err = iouring_syscall.IOURingEnter(iour.fd, uint32(submitted), waitCount, flags, nil)
	return
}
*/

// CancelRequest by request id
func (iour *IOURing) CancelRequest(id uint64, ch chan<- *Result) error {
	_, err := iour.SubmitRequest(cancelRequest(id), ch)
	return err
}

func (iour *IOURing) getCQEvent(wait bool) (cqe *iouring_syscall.CompletionQueueEvent, err error) {
	for {
		if cqe = iour.cq.peek(); cqe != nil {
			iour.cq.advance(1)
			return
		}

		if !wait && !iour.sq.cqOverflow() {
			err = syscall.EAGAIN
			return
		}

		runtime.Gosched()

		/*
			_, err = iouring_syscall.IOURingEnter(iour.fd, 0, 1, iouring_syscall.IORING_ENTER_FLAGS_GETEVENTS, nil)
			if err != nil {
				return
			}
		*/
	}
}

func (iour *IOURing) run() {
	for {
		if atomic.LoadInt64(&iour.submits) <= 0 {
			<-iour.submitSign
			continue
		}

		cqe, err := iour.getCQEvent(true)
		if cqe == nil || err != nil {
			log.Println("runComplete error: ", err)
			continue
		}
		atomic.AddInt64(&iour.submits, -1)

		// log.Println("cqe user data", (cqe.UserData))

		iour.userDataLock.Lock()
		userData := iour.userDatas[cqe.UserData]
		if userData == nil {
			iour.userDataLock.Unlock()
			log.Println("runComplete: notfound user data ", uintptr(cqe.UserData))
			continue
		}
		delete(iour.userDatas, cqe.UserData)
		iour.userDataLock.Unlock()

		// ignore link timeout
		if userData.opcode == iouring_syscall.IORING_OP_LINK_TIMEOUT {
			continue
		}

		userData.result.load(cqe)

		if userData.done != nil {
			userData.done <- userData.result
		}
	}
}

func cancelRequest(id uint64) Request {
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = cancelResolver
		sqe.PrepOperation(iouring_syscall.IORING_OP_ASYNC_CANCEL, -1, id, 0, 0)
	}
}
