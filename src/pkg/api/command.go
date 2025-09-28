package api

import (
	"fmt"

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

func (c *Command) BuildMessage() (modbus.MessageData, error) {
	relayIndex := c.RelayIndex()
	switch c.Command {
	case RelayCommandOn:
		return modbus.NewWriteSingleCoil(relayIndex, modbus.WriteCommandOn), nil
	case RelayCommandOff:
		return modbus.NewWriteSingleCoil(relayIndex, modbus.WriteCommandOff), nil
	case RelayCommandToggle:
		return modbus.NewWriteSingleCoil(relayIndex, modbus.WriteCommandToggle), nil
	default:
		return nil, fmt.Errorf("unknown command: %s", c.Command)
	}
}
