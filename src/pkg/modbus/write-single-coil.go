package modbus

import (
	"encoding/binary"

	"github.com/jakerobb/modbus-eth-controller/pkg/util"
)

type WriteSingleCoil struct {
	MessageHeader *MessageHeader
	FunctionCode  byte
	Relay         uint16
	Command       WriteCommand
	data          util.HexBytes
}

type WriteCommand uint16

const (
	WriteCommandOn     WriteCommand = 0xFF00
	WriteCommandOff    WriteCommand = 0x0000
	WriteCommandToggle WriteCommand = 0x5500
)

func NewWriteSingleCoil(relay int, command WriteCommand) *WriteSingleCoil {
	return &WriteSingleCoil{
		FunctionCode: byte(WriteSingleCoilFunction),
		Relay:        uint16(relay),
		Command:      command,
	}
}

func (w *WriteSingleCoil) ToDataBytes() util.HexBytes {
	if len(w.data) == 0 {
		msg := make([]byte, 5)
		msg[0] = w.FunctionCode
		binary.BigEndian.PutUint16(msg[1:], w.Relay)
		binary.BigEndian.PutUint16(msg[3:], uint16(w.Command))
		w.data = msg
	}
	return w.data
}

func (w *WriteSingleCoil) ValidateResponse(request *Message, response *Response) error {
	return ValidateEchoResponse(request.ToBytes(), response)
}

func (w *WriteSingleCoil) ParseResponse(_ *Response) (interface{}, error) {
	return nil, nil
}
