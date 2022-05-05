//go:build linux
// +build linux

package iouring

import (
	"sync/atomic"

	iouring_syscall "github.com/iceber/iouring-go/syscall"
)

// iouring operations
const (
	OpNop uint8 = iota
	OpReadv
	OpWritev
	OpFsync
	OpReadFixed
	OpWriteFixed
	OpPollAdd
	OpPollRemove
	OpSyncFileRange
	OpSendmsg
	OpRecvmsg
	OpTimeout
	OpTimeoutRemove
	OpAccept
	OpAsyncCancel
	OpLinkTimeout
	OpConnect
	OpFallocate
	OpOpenat
	OpClose
	OpFilesUpdate
	OpStatx
	OpRead
	OpWrite
	OpFadvise
	OpMadvise
	OpSend
	OpRecv
	OpOpenat2
	OpEpollCtl
	OpSplice
	OpProvideBuffers
	OpRemoveBuffers
	OpTee
	OpShutdown
	OpUnlinkat
	OpMkdirat
)

// cancel operation return value
const (
	RequestCanceledSuccessfully = 0
	RequestMaybeCanceled        = 1
)

// timeout operation return value
const (
	TimeoutExpiration = 0
	CountCompletion   = 1
)

var _zero uintptr

type SubmissionQueue struct {
	ptr  uintptr
	size uint32

	head    *uint32
	tail    *uint32
	mask    *uint32
	entries *uint32 // specifies the number of submission queue ring entries
	flags   *uint32 // used by the kernel to communicate stat information to the application
	dropped *uint32 // incrementd for each invalid submission queue entry encountered in the ring buffer

	array []uint32
	sqes  []iouring_syscall.SubmissionQueueEntry // submission queue ring

	sqeHead uint32
	sqeTail uint32
}

func (queue *SubmissionQueue) getSQEntry() *iouring_syscall.SubmissionQueueEntry {
	head := atomic.LoadUint32(queue.head)
	next := queue.sqeTail + 1

	if (next - head) <= *queue.entries {
		sqe := &queue.sqes[queue.sqeTail&*queue.mask]
		queue.sqeTail = next
		sqe.Reset()
		return sqe
	}
	return nil
}

func (queue *SubmissionQueue) fallback(i uint32) {
	queue.sqeTail -= i
}

func (queue *SubmissionQueue) cqOverflow() bool {
	return (atomic.LoadUint32(queue.flags) & iouring_syscall.IORING_SQ_CQ_OVERFLOW) != 0
}

func (queue *SubmissionQueue) needWakeup() bool {
	return (atomic.LoadUint32(queue.flags) & iouring_syscall.IORING_SQ_NEED_WAKEUP) != 0
}

// sync internal status with kernel ring state on the SQ side
// return the number of pending items in the SQ ring, for the shared ring.
func (queue *SubmissionQueue) flush() int {
	if queue.sqeHead == queue.sqeTail {
		return int(*queue.tail - *queue.head)
	}

	tail := *queue.tail
	for toSubmit := queue.sqeTail - queue.sqeHead; toSubmit > 0; toSubmit-- {
		queue.array[tail&*queue.mask] = queue.sqeHead & *queue.mask
		tail++
		queue.sqeHead++
	}

	atomic.StoreUint32(queue.tail, tail)
	return int(tail - *queue.head)
}

type CompletionQueue struct {
	ptr  uintptr
	size uint32

	head     *uint32
	tail     *uint32
	mask     *uint32
	overflow *uint32
	entries  *uint32
	flags    *uint32

	cqes []iouring_syscall.CompletionQueueEvent
}

func (queue *CompletionQueue) peek() (cqe *iouring_syscall.CompletionQueueEvent) {
	head := *queue.head
	if head != atomic.LoadUint32(queue.tail) {
		//	if head < atomic.LoadUint32(queue.tail) {
		cqe = &queue.cqes[head&*queue.mask]
	}
	return
}

func (queue *CompletionQueue) advance(num uint32) {
	if num != 0 {
		atomic.AddUint32(queue.head, num)
	}
}
