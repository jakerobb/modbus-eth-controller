package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

type Command struct {
	Command string `json:"command"`
	Relay   int    `json:"relay,omitempty"`
}

func (c *Command) Send(conn net.Conn) ([]byte, error) {
	msg, err := c.getBytes()
	if err != nil {
		return nil, err
	}

	// Send the command over the TCP connection
	_, err = conn.Write(msg)
	if err != nil {
		return nil, err
	}

	time.Sleep(5 * time.Millisecond) // slight delay to avoid overwhelming the device
	return msg, nil
}

func (c *Command) getBytes() ([]byte, error) {
	// Construct the 12-byte Modbus TCP frame
	// 0-1 Transaction ID: 0x0000
	// 2-3 Protocol ID: 0x0000
	// 4-5 Length: number of bytes past this (8 + Len(commands) * 4)
	// 6 Unit ID: 0x01
	// 7 Function Code: 0x05 (Write Single Coil)
	// 8-9 Address: relay number (big endian)
	// 10-11 Data: 0xFF00 for ON, 0x0000 for OFF

	msg := make([]byte, 12)
	binary.BigEndian.PutUint16(msg[0:], uint16(0))
	binary.BigEndian.PutUint16(msg[2:], uint16(0))
	binary.BigEndian.PutUint16(msg[4:], uint16(12))
	msg[6] = 0x01
	msg[7] = 0x05

	binary.BigEndian.PutUint16(msg[8:], uint16(c.Relay))

	var err error
	switch c.Command {
	case "on":
		binary.BigEndian.PutUint16(msg[10:], 0xFF00)
	case "off":
		binary.BigEndian.PutUint16(msg[10:], 0x0000)
	default:
		err = errors.New(fmt.Sprintf("invalid command %s", c.Command))
	}
	return msg, err
}
