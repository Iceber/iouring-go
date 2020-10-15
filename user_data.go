// +build linux

package iouring

import (
	"unsafe"
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

// TODO(iceber): use sync.Poll
func makeUserData(iour *IOURing, ch chan<- *Result) *UserData {
	userData := &UserData{
		resulter: ch,
		result:   &Result{iour: iour, done: make(chan struct{})},
	}

	userData.id = uint64(uintptr(unsafe.Pointer(userData)))
	userData.result.id = userData.id
	return userData
}
