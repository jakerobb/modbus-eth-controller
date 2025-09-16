package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
)

type CoilStates struct {
	Coils map[string]bool `json:"coils"`
}

type ReadCoils struct {
	MessageHeader *MessageHeader
	FunctionCode  byte
	StartAddress  uint16
	Quantity      uint16
}

func NewReadCoils(startAddress, quantity uint16) *ReadCoils {
	return &ReadCoils{
		FunctionCode: byte(ReadCoilsFunction),
		StartAddress: startAddress,
		Quantity:     quantity,
	}
}

func (w *ReadCoils) ToDataBytes() []byte {
	msg := make([]byte, 5)
	msg[0] = w.FunctionCode
	binary.BigEndian.PutUint16(msg[1:], w.StartAddress)
	binary.BigEndian.PutUint16(msg[3:], w.Quantity)
	return msg
}

func (w *ReadCoils) ValidateResponse(msg *Message, response *Response) error {
	responseData := response.Data

	validationErrors := make([]error, 0)
	if response.MessageHeader.TransactionID != msg.Header.TransactionID {
		validationErrors = append(validationErrors, fmt.Errorf("response transaction ID %x does not match request transaction ID %x", response.MessageHeader.TransactionID, msg.Header.TransactionID))
	}
	if err := validateFunctionCode(responseData[1], ReadCoilsFunction); err != nil {
		validationErrors = append(validationErrors, err)
	}
	if len(responseData) != 3 {
		validationErrors = append(validationErrors, fmt.Errorf("response data length is %d, expected 4", len(responseData)))
	}

	if len(validationErrors) == 0 {
		return nil
	}
	return errors.Join(validationErrors...)
}

func (w *ReadCoils) ParseResponse(response *Response) (interface{}, error) {
	result := make(map[string]bool)
	responseData := response.Data

	for i := 0; i < int(w.Quantity); i++ {
		byteIndex := 2 + (i / 8)
		bitIndex := i % 8
		result[strconv.Itoa(i+1)] = (responseData[byteIndex] & (1 << bitIndex)) != 0
	}
	return &CoilStates{
		Coils: result,
	}, nil
}
