package modbus

type FunctionCode byte

const (
	ReadCoilsFunction       FunctionCode = 0x01
	WriteSingleCoilFunction FunctionCode = 0x05
)
