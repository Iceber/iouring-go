// +build linux

package iouring

import (
	"errors"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"

	iouring_syscall "github.com/iceber/iouring-go/syscall"
)

type Request func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData)

func (iour *IOURing) Read(file *os.File, b []byte, ch chan<- *Result) (uint64, error) {
	fd := int(file.Fd())
	if fd < 0 {
		return 0, errors.New("invalid file")
	}

	return iour.SubmitRequest(Read(fd, b), ch)
}

func (iour *IOURing) Write(file *os.File, b []byte, ch chan<- *Result) (uint64, error) {
	fd := int(file.Fd())
	if fd < 0 {
		return 0, errors.New("invalid file")
	}

	return iour.SubmitRequest(Write(fd, b), ch)
}

func (iour *IOURing) Pread(file *os.File, b []byte, offset uint64, ch chan<- *Result) (uint64, error) {
	fd := int(file.Fd())
	if fd < 0 {
		return 0, errors.New("invalid file")
	}

	return iour.SubmitRequest(Pread(fd, b, offset), ch)
}

func (iour *IOURing) Pwrite(file *os.File, b []byte, offset uint64, ch chan<- *Result) (uint64, error) {
	fd := int(file.Fd())
	if fd < 0 {
		return 0, errors.New("invalid file")
	}

	return iour.SubmitRequest(Pwrite(fd, b, offset), ch)
}

func RequestWithInfo(request Request, info interface{}) Request {
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		request(sqe, userData)
		userData.SetRequestInfo(info)
	}
}

func Nop() Request {
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		sqe.PrepOperation(iouring_syscall.IORING_OP_NOP, -1, 0, 0, 0)
	}
}

func Read(fd int, b []byte) Request {
	var bp unsafe.Pointer
	if len(b) > 0 {
		bp = unsafe.Pointer(&b[0])
	} else {
		bp = unsafe.Pointer(&_zero)
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = fdResolver
		userData.SetRequestBuffer(&b, nil)

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_READ,
			int32(fd),
			uint64(uintptr(bp)),
			uint32(len(b)),
			0,
		)
	}
}

func Pread(fd int, b []byte, offset uint64) Request {
	var bp unsafe.Pointer
	if len(b) > 0 {
		bp = unsafe.Pointer(&b[0])
	} else {
		bp = unsafe.Pointer(&_zero)
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = fdResolver
		userData.SetRequestBuffer(&b, nil)

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_READ,
			int32(fd),
			uint64(uintptr(bp)),
			uint32(len(b)),
			uint64(offset),
		)
	}
}

func Write(fd int, b []byte) Request {
	var bp unsafe.Pointer
	if len(b) > 0 {
		bp = unsafe.Pointer(&b[0])
	} else {
		bp = unsafe.Pointer(&_zero)
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = fdResolver
		userData.SetRequestBuffer(&b, nil)

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_WRITE,
			int32(fd),
			uint64(uintptr(bp)),
			uint32(len(b)),
			0,
		)
	}
}

func Pwrite(fd int, b []byte, offset uint64) Request {
	var bp unsafe.Pointer
	if len(b) > 0 {
		bp = unsafe.Pointer(&b[0])
	} else {
		bp = unsafe.Pointer(&_zero)
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = fdResolver
		userData.SetRequestBuffer(&b, nil)

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_WRITE,
			int32(fd),
			uint64(uintptr(bp)),
			uint32(len(b)),
			uint64(offset),
		)
	}
}

func Readv(fd int, bs [][]byte) Request {
	iovecs := bytes2iovec(bs)

	var bp unsafe.Pointer
	if len(iovecs) > 0 {
		bp = unsafe.Pointer(&iovecs[0])
	} else {
		bp = unsafe.Pointer(&_zero)
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = fdResolver
		userData.SetRequestBuffers(&bs)

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_READV,
			int32(fd),
			uint64(uintptr(bp)),
			uint32(len(iovecs)),
			0,
		)
	}
}

func Preadv(fd int, bs [][]byte, offset uint64) Request {
	iovecs := bytes2iovec(bs)

	var bp unsafe.Pointer
	if len(iovecs) > 0 {
		bp = unsafe.Pointer(&iovecs[0])
	} else {
		bp = unsafe.Pointer(&_zero)
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = fdResolver
		userData.SetRequestBuffers(&bs)

		sqe.PrepOperation(iouring_syscall.IORING_OP_READV,
			int32(fd),
			uint64(uintptr(bp)),
			uint32(len(iovecs)),
			offset,
		)
	}
}

func Writev(fd int, bs [][]byte) Request {
	iovecs := bytes2iovec(bs)

	var bp unsafe.Pointer
	if len(iovecs) > 0 {
		bp = unsafe.Pointer(&iovecs[0])
	} else {
		bp = unsafe.Pointer(&_zero)
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = fdResolver
		userData.SetRequestBuffers(&bs)

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_WRITEV,
			int32(fd),
			uint64(uintptr(bp)),
			uint32(len(iovecs)),
			0,
		)
	}
}

func Pwritev(fd int, bs [][]byte, offset int64) Request {
	iovecs := bytes2iovec(bs)

	var bp unsafe.Pointer
	if len(iovecs) > 0 {
		bp = unsafe.Pointer(&iovecs[0])
	} else {
		bp = unsafe.Pointer(&_zero)
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = fdResolver
		userData.SetRequestBuffers(&bs)

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_WRITEV,
			int32(fd),
			uint64(uintptr(bp)),
			uint32(len(iovecs)),
			uint64(offset),
		)
	}
}

func Send(sockfd int, b []byte, flags int) Request {
	var bp unsafe.Pointer
	if len(b) > 0 {
		bp = unsafe.Pointer(&b[0])
	} else {
		bp = unsafe.Pointer(&_zero)
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.SetRequestBuffer(&b, nil)

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_SEND,
			int32(sockfd),
			uint64(uintptr(bp)),
			uint32(len(b)),
			0,
		)
		sqe.SetOpFlags(uint32(flags))
	}
}

func Recv(sockfd int, b []byte, flags int) Request {
	var bp unsafe.Pointer
	if len(b) > 0 {
		bp = unsafe.Pointer(&b[0])
	} else {
		bp = unsafe.Pointer(&_zero)
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.SetRequestBuffer(&b, nil)

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_RECV,
			int32(sockfd),
			uint64(uintptr(bp)),
			uint32(len(b)),
			0,
		)
		sqe.SetOpFlags(uint32(flags))
	}
}

func Sendmsg(sockfd int, p, oob []byte, to syscall.Sockaddr, flags int) (Request, error) {
	var ptr unsafe.Pointer
	var salen uint32
	if to != nil {
		var err error
		ptr, salen, err = sockaddr(to)
		if err != nil {
			return nil, err
		}
	}

	msg := &syscall.Msghdr{}
	msg.Name = (*byte)(ptr)
	msg.Namelen = uint32(salen)
	var iov syscall.Iovec
	if len(p) > 0 {
		iov.Base = &p[0]
		iov.SetLen(len(p))
	}
	var dummy byte
	if len(oob) > 0 {
		if len(p) == 0 {
			var sockType int
			sockType, err := syscall.GetsockoptInt(sockfd, syscall.SOL_SOCKET, syscall.SO_TYPE)
			if err != nil {
				return nil, err
			}
			// send at least one normal byte
			if sockType != syscall.SOCK_DGRAM {
				iov.Base = &dummy
				iov.SetLen(1)
			}
		}
		msg.Control = &oob[0]
		msg.SetControllen(len(oob))
	}
	msg.Iov = &iov
	msg.Iovlen = 1

	resolver := func(result *Result) {
		result.r0 = int(result.res)
		errResolver(result)
		if result.err != nil {
			return
		}

		if len(oob) > 0 && len(p) == 0 {
			result.r0 = 0
		}
	}

	msgptr := unsafe.Pointer(msg)
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.hold(msg, to)
		userData.result.resolver = resolver
		userData.SetRequestBuffer(&p, &oob)

		sqe.PrepOperation(iouring_syscall.IORING_OP_SENDMSG, int32(sockfd), uint64(uintptr(msgptr)), 1, 0)
		sqe.SetOpFlags(uint32(flags))
	}, nil
}

func Recvmsg(sockfd int, p, oob []byte, to syscall.Sockaddr, flags int) (Request, error) {
	var msg syscall.Msghdr
	var rsa syscall.RawSockaddrAny
	msg.Name = (*byte)(unsafe.Pointer(&rsa))
	msg.Namelen = uint32(syscall.SizeofSockaddrAny)
	var iov syscall.Iovec
	if len(p) > 0 {
		iov.Base = &p[0]
		iov.SetLen(len(p))
	}
	var dummy byte
	if len(oob) > 0 {
		if len(p) == 0 {
			var sockType int
			sockType, err := syscall.GetsockoptInt(sockfd, syscall.SOL_SOCKET, syscall.SO_TYPE)
			if err != nil {
				return nil, err
			}
			// receive at least one normal byte
			if sockType != syscall.SOCK_DGRAM {
				iov.Base = &dummy
				iov.SetLen(1)
			}
		}
		msg.Control = &oob[0]
		msg.SetControllen(len(oob))
	}
	msg.Iov = &iov
	msg.Iovlen = 1

	resolver := func(result *Result) {
		result.r0 = int(result.res)
		errResolver(result)
		if result.err != nil {
			return
		}

		if len(oob) > 0 && len(p) == 0 {
			result.r0 = 0
		}
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.hold(&msg, &rsa)
		userData.result.resolver = resolver
		userData.SetRequestBuffer(&p, &oob)

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_RECVMSG,
			int32(sockfd),
			uint64(uintptr(unsafe.Pointer(&msg))),
			1,
			0,
		)
		sqe.SetOpFlags(uint32(flags))
	}, nil
}

func Accept(sockfd int) Request {
	var rsa syscall.RawSockaddrAny
	var len uint32 = syscall.SizeofSockaddrAny

	resolver := func(result *Result) {
		fd := int(result.res)
		errResolver(result)
		if result.err != nil {
			return
		}

		result.r0 = fd
		result.r1, result.err = anyToSockaddr(&rsa)
		if result.err != nil {
			syscall.Close(fd)
			result.r0 = 0
		}
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.hold(&len)
		userData.result.resolver = resolver
		sqe.PrepOperation(iouring_syscall.IORING_OP_ACCEPT, int32(sockfd), uint64(uintptr(unsafe.Pointer(&rsa))), 0, uint64(uintptr(unsafe.Pointer(&len))))
	}
}

func Accept4(sockfd int, flags int) Request {
	var rsa syscall.RawSockaddrAny
	var len uint32 = syscall.SizeofSockaddrAny

	resolver := func(result *Result) {
		fd := int(result.res)
		errResolver(result)
		if result.err != nil {
			return
		}

		if len > syscall.SizeofSockaddrAny {
			panic("RawSockaddrAny too small")
		}

		result.r0 = fd
		result.r1, result.err = anyToSockaddr(&rsa)
		if result.err != nil {
			syscall.Close(fd)
			result.r0 = 0
		}
	}
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.hold(&rsa, &len)
		userData.result.resolver = resolver

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_ACCEPT,
			int32(sockfd),
			uint64(uintptr(unsafe.Pointer(&rsa))),
			0,
			uint64(uintptr(unsafe.Pointer(&len))),
		)
		sqe.SetOpFlags(uint32(flags))
	}
}

func Connect(sockfd int, sa syscall.Sockaddr) (Request, error) {
	ptr, n, err := sockaddr(sa)
	if err != nil {
		return nil, err
	}

	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.hold(sa)
		userData.result.resolver = errResolver

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_CONNECT,
			int32(sockfd),
			uint64(uintptr(ptr)),
			0,
			uint64(n),
		)
	}, nil
}

func Openat(dirfd int, path string, flags uint32, mode uint32) (Request, error) {
	flags |= syscall.O_LARGEFILE
	b, err := syscall.ByteSliceFromString(path)
	if err != nil {
		return nil, err
	}

	bp := unsafe.Pointer(&b[0])
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.hold(&b)
		userData.result.resolver = fdResolver

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_OPENAT,
			int32(dirfd),
			uint64(uintptr(bp)),
			mode,
			0,
		)
		sqe.SetOpFlags(flags)
	}, nil
}

func Openat2(dirfd int, path string, how *unix.OpenHow) (Request, error) {
	b, err := syscall.ByteSliceFromString(path)
	if err != nil {
		return nil, err
	}

	bp := unsafe.Pointer(&b[0])
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.hold(&b)
		userData.result.resolver = fdResolver

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_OPENAT2,
			int32(dirfd),
			uint64(uintptr(bp)),
			unix.SizeofOpenHow,
			uint64(uintptr(unsafe.Pointer(how))),
		)
	}, nil
}

func Statx(dirfd int, path string, flags uint32, mask int, stat *unix.Statx_t) (Request, error) {
	b, err := syscall.ByteSliceFromString(path)
	if err != nil {
		return nil, err
	}

	bp := unsafe.Pointer(&b[0])
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = errResolver
		userData.hold(&b, stat)

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_STATX,
			int32(dirfd),
			uint64(uintptr(bp)),
			uint32(mask),
			uint64(uintptr(unsafe.Pointer(stat))),
		)
		sqe.SetOpFlags(flags)
	}, nil
}

func Fsync(fd int) Request {
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = errResolver
		sqe.PrepOperation(iouring_syscall.IORING_OP_FSYNC, int32(fd), 0, 0, 0)
	}
}

func Fdatasync(fd int) Request {
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = errResolver
		sqe.PrepOperation(iouring_syscall.IORING_OP_FSYNC, int32(fd), 0, 0, 0)
		sqe.SetOpFlags(iouring_syscall.IORING_FSYNC_DATASYNC)
	}
}

func Fallocate(fd int, mode uint32, off int64, length int64) Request {
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = errResolver

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_FALLOCATE,
			int32(fd),
			uint64(length),
			uint32(mode),
			uint64(off),
		)
	}
}

func Close(fd int) Request {
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = errResolver
		sqe.PrepOperation(iouring_syscall.IORING_OP_CLOSE, int32(fd), 0, 0, 0)
	}
}

func Madvise(b []byte, advice int) Request {
	var bp unsafe.Pointer
	if len(b) > 0 {
		bp = unsafe.Pointer(&b[0])
	} else {
		bp = unsafe.Pointer(&_zero)
	}
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = errResolver
		userData.SetRequestBuffer(&b, nil)

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_MADVISE,
			-1,
			uint64(uintptr(bp)),
			uint32(len(b)),
			0,
		)
		sqe.SetOpFlags(uint32(advice))
	}
}

func EpollCtl(epfd int, op int, fd int, event *syscall.EpollEvent) Request {
	return func(sqe *iouring_syscall.SubmissionQueueEntry, userData *UserData) {
		userData.result.resolver = errResolver

		sqe.PrepOperation(
			iouring_syscall.IORING_OP_EPOLL_CTL,
			int32(epfd),
			uint64(uintptr(unsafe.Pointer(event))),
			uint32(op),
			uint64(fd),
		)
	}
}
