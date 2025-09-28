package modbus

import (
	"context"
	"encoding/binary"
	"io"
	"net"

	"github.com/jakerobb/modbus-eth-controller/pkg/util"
)

type Response struct {
	MessageHeader *MessageHeader
	Data          util.HexBytes
}

func (r Response) ToBytes() util.HexBytes {
	msg := make([]byte, 7+len(r.Data))
	binary.BigEndian.PutUint16(msg[0:], r.MessageHeader.TransactionID)
	binary.BigEndian.PutUint16(msg[2:], r.MessageHeader.ProtocolID)
	binary.BigEndian.PutUint16(msg[4:], r.MessageHeader.Length)
	msg[6] = r.MessageHeader.UnitID
	copy(msg[7:], r.Data)
	return msg
}

func ReadResponse(ctx context.Context, conn net.Conn) (*Response, error) {
	// read the 7-byte header first; it tells us the full message length
	var header util.HexBytes = make([]byte, 7)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, err
	}

	messageHeader := &MessageHeader{
		TransactionID: binary.BigEndian.Uint16(header[0:2]),
		ProtocolID:    binary.BigEndian.Uint16(header[2:4]),
		Length:        binary.BigEndian.Uint16(header[4:6]),
		UnitID:        header[6],
	}

	// subtract one here because we consider the UnitID to be part of the header, but modbus protocol counts it in the length
	var payload util.HexBytes = make([]byte, messageHeader.Length-1)
	_, err = io.ReadFull(conn, payload)
	if err != nil {
		return nil, err
	}

	util.LogDebug(ctx, "Response", "header", header, "payload", payload)

	return &Response{
		MessageHeader: messageHeader,
		Data:          payload,
	}, nil
}
