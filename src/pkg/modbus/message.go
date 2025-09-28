package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/jakerobb/modbus-eth-controller/pkg/util"
)

var lastTransactionId uint16 = 0

// Message represents a Modbus TCP message frame.
// Modbus TCP frame structure:
// 0-1 Transaction ID: value that will be echoed back in the response
// 2-3 Protocol ID: 0x0000
// 4-5 Length: number of bytes remaining in this message
// 7 Unit ID: always 0x01
// 7-X: Data

type Message struct {
	Header *MessageHeader
	Data   util.HexBytes
	Bytes  util.HexBytes
}
type MessageHeader struct {
	TransactionID uint16
	ProtocolID    uint16
	Length        uint16
	UnitID        byte
	Bytes         util.HexBytes
}

func (mh MessageHeader) ToBytes() util.HexBytes {
	if len(mh.Bytes) == 0 {
		msg := make([]byte, 7)
		binary.BigEndian.PutUint16(msg[0:], mh.TransactionID)
		binary.BigEndian.PutUint16(msg[2:], mh.ProtocolID)
		binary.BigEndian.PutUint16(msg[4:], mh.Length)
		msg[6] = mh.UnitID
		mh.Bytes = msg
	}
	return mh.Bytes
}

type MessageData interface {
	ToDataBytes() util.HexBytes
	ValidateResponse(request *Message, response *Response) error
	ParseResponse(response *Response) (interface{}, error)
}

func NextTransactionId() uint16 {
	lastTransactionId = lastTransactionId + 1
	return lastTransactionId
}

func createMessage(data MessageData) *Message {
	dataBytes := data.ToDataBytes()

	header := &MessageHeader{
		TransactionID: NextTransactionId(),
		ProtocolID:    0,
		Length:        uint16(len(dataBytes) + 1),
		UnitID:        1,
	}

	return &Message{
		Header: header,
		Data:   dataBytes,
	}
}

func (m *Message) ToBytes() util.HexBytes {
	if len(m.Bytes) == 0 {
		mh := m.Header
		headerBytes := mh.ToBytes()
		data := m.Data
		msg := make([]byte, len(headerBytes)+len(data))
		copy(msg[0:], headerBytes)
		copy(msg[7:], data)
		m.Bytes = msg
	}
	return m.Bytes
}

func (m *Message) sendMessage(ctx context.Context, conn net.Conn) (*Response, error) {
	messageBytes := m.ToBytes()
	util.LogDebug(ctx, "Sending", "header", m.Header.ToBytes(), "payload", m.Data)
	_, err := conn.Write(messageBytes)
	if err != nil {
		return nil, err
	}

	return ReadResponse(ctx, conn)
}

func Send(ctx context.Context, conn net.Conn, messageData MessageData) (*Message, interface{}, error) {
	msg := createMessage(messageData)
	response, err := msg.sendMessage(ctx, conn)
	if err != nil {
		return nil, nil, err
	}

	if response.MessageHeader.Length != uint16(len(response.Data)+1) {
		return nil, nil, fmt.Errorf("response length mismatch: header length %d, actual data length %d", response.MessageHeader.Length, len(response.Data))
	}

	if err = checkForException(response); err != nil {
		util.LogDebug(ctx, "Got an exception response!", "error", err.Error())
		return nil, nil, err
	}

	err = messageData.ValidateResponse(msg, response)
	if err != nil {
		return nil, nil, err
	}
	util.LogDebug(ctx, "Response is valid")
	parseResponse, err := messageData.ParseResponse(response)
	return msg, parseResponse, err
}

func validateFunctionCode(actual byte, expected FunctionCode) error {
	if FunctionCode(actual) != expected {
		return fmt.Errorf("unexpected function code: %02X; expected %02X", actual, expected)
	}
	return nil
}

func checkForException(response *Response) error {
	fc := response.Data[0]
	if fc&0x80 != 0 {
		exceptionCode := response.Data[1]
		return &ModbusError{
			Code:     exceptionCode,
			Function: fc,
		}
	}
	return nil
}
