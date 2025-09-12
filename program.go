package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type Program struct {
	Address               string      `json:"address"`
	CommandIntervalMillis int         `json:"commandIntervalMillis,omitempty"`
	Loops                 int         `json:"loops,omitempty"`
	Commands              [][]Command `json:"commands"`
	Debug                 bool        `json:"debug,omitempty"`
}

func ParseProgram(programBytes []byte) (*Program, error) {
	var program Program
	if err := json.Unmarshal(programBytes, &program); err != nil {
		return nil, fmt.Errorf("failed to parse JSON program: %w", err)
	}

	if program.Address == "" {
		return nil, fmt.Errorf("missing required field: address")
	}
	return &program, nil
}

func (p *Program) Run() error {
	// Establish TCP connection
	p.logDebug("Connecting to %s\n", p.Address)
	conn, err := net.Dial("tcp", p.Address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", p.Address, err)
	}
	p.logDebug("Connected to %s\n", p.Address)
	defer conn.Close()

	loops := p.Loops
	if loops <= 0 {
		loops = 1
	}

	p.logDebug("Parsed program: %+v\n", p)

	p.logDebug("Starting command execution for %d loops\n", loops)
	for i := 0; i < loops; i++ {
		p.logDebug("Loop %d/%d\n", i+1, loops)
		// Execute each command in order
		for j, cmdGroup := range p.Commands {
			p.logDebug("  Executing command group %d: %+v\n", j+1, cmdGroup)
			// Execute commands in the group sequentially
			for k, cmd := range cmdGroup {
				p.logDebug("    Executing command %d: %+v\n", k+1, cmd)
				// Send command to the device
				msg, err := cmd.Send(conn)
				if err != nil {
					return fmt.Errorf("failed to send commands: %w", err)
				} else {
					p.logDebug("      Sent message % X\n", msg)
				}
			}

			// Wait for the specified interval before the next command group
			if p.CommandIntervalMillis > 0 {
				p.logDebug("  Waiting for %d milliseconds before next command group\n", p.CommandIntervalMillis)
				time.Sleep(time.Duration(p.CommandIntervalMillis) * time.Millisecond)
			}
		}
	}
	return nil
}

func (p *Program) logDebug(format string, a ...any) {
	if p.Debug {
		fmt.Printf(format, a...)
	}
}
