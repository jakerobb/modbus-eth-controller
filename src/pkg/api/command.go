package api

import (
	"fmt"
	"os"

	"github.com/jakerobb/modbus-eth-controller/pkg/modbus"
)

type RelayCommand string

const (
	RelayCommandOn     RelayCommand = "on"
	RelayCommandOff    RelayCommand = "off"
	RelayCommandToggle RelayCommand = "toggle"
)

type Command struct {
	Command RelayCommand `json:"command" example:"toggle"`
	Relay   int          `json:"relay" example:"1"`
}

func (c *Command) RelayIndex() int {
	return c.Relay - 1
}

func (c *Command) BuildMessage() modbus.MessageData {
	relayIndex := c.RelayIndex()
	switch c.Command {
	case RelayCommandOn:
		return modbus.NewWriteSingleCoil(relayIndex, modbus.WriteCommandOn)
	case RelayCommandOff:
		return modbus.NewWriteSingleCoil(relayIndex, modbus.WriteCommandOff)
	case RelayCommandToggle:
		return modbus.NewWriteSingleCoil(relayIndex, modbus.WriteCommandToggle)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "Unknown command: %s\n", c.Command)
		return nil
	}
}
