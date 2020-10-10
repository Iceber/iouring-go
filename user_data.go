// +build linux

package iouring

type UserData struct {
	done   chan<- *Result
	opcode uint8

	holds  []interface{}
	result *Result
}

func (data *UserData) SetResultResolver(resolver ResultResolver) {
	data.result.resolver = resolver
}

func (data *UserData) SetRequestInfo(info interface{}) {
	data.result.requestInfo = info
}

func (data *UserData) hold(vars ...interface{}) {
	data.holds = vars
}

func (data *UserData) Hold(vars ...interface{}) {
	data.holds = append(data.holds, vars)
}

func (data *UserData) SetRequestBuffer(b0, b1 *[]byte) {
	data.result.b0, data.result.b1 = b0, b1
}

func (data *UserData) SetRequestBuffers(bs *[][]byte) {
	data.result.bs = bs
}

func (data *UserData) setOpcode(opcode uint8) {
	data.opcode = opcode
	data.result.opcode = opcode
}

func makeUserData(ch chan<- *Result) *UserData {
	return &UserData{
		done:   ch,
		result: &Result{},
	}
}
