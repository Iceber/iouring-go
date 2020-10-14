// +build linux

package iouring

import (
	"unsafe"

	iouring_syscall "github.com/iceber/iouring-go/syscall"
)

type UserData struct {
	id uint64

	resulter chan<- *Result
	opcode   uint8

	holds  []interface{}
	result *Result
}

func (data *UserData) SetResultResolver(resolver ResultResolver) {
	data.result.resolver = resolver
}

func (data *UserData) SetRequestInfo(info interface{}) {
	data.result.requestInfo = info
}

func (data *UserData) SetRequestBuffer(b0, b1 *[]byte) {
	data.result.b0, data.result.b1 = b0, b1
}

func (data *UserData) SetRequestBuffers(bs *[][]byte) {
	data.result.bs = bs
}

func (data *UserData) Hold(vars ...interface{}) {
	data.holds = append(data.holds, vars)
}

func (data *UserData) hold(vars ...interface{}) {
	data.holds = vars
}

func (data *UserData) setOpcode(opcode uint8) {
	data.opcode = opcode
	data.result.opcode = opcode
}

func (data *UserData) getResult(cqe *iouring_syscall.CompletionQueueEvent) *Result {
	data.result.id = data.id
	data.result.res = cqe.Result

	return data.result
}

func makeUserData(ch chan<- *Result) *UserData {
	userData := &UserData{
		resulter: ch,
		result:   &Result{},
	}

	userData.id = uint64(uintptr(unsafe.Pointer(userData)))
	return userData
}
