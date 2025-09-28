package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jakerobb/modbus-eth-controller/pkg/modbus"
	"github.com/jakerobb/modbus-eth-controller/pkg/util"
)

var slugifyRegexp = regexp.MustCompile(`[^a-z0-9]+`)

type ProgramRequest struct {
	Address               string      `json:"address" example:"modbus.lan:4196"`
	Commands              [][]Command `json:"commands"`
	Loops                 int         `json:"loops,omitempty" example:"2"`
	CommandIntervalMillis int         `json:"commandIntervalMillis,omitempty" example:"200"`
	Debug                 bool        `json:"debug,omitempty" example:"true"`
}

type Program struct {
	ProgramRequest
	Slug         string     `json:"slug,omitempty" example:"doorbell"`
	Path         string     `json:"path,omitempty" example:"/etc/modbus/doorbell.json"`
	LastModified *time.Time `json:"lastModified,omitempty" example:"2025-09-14T12:00:00Z"`
}

func ParseProgramFromFile(path string) (*Program, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	program, err := ParseProgram(data)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	modTime := info.ModTime()
	program.LastModified = &modTime
	program.Slug = SlugifyFilename(path)
	program.Path = path
	return program, nil
}

func SlugifyFilename(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	name = strings.ToLower(name)
	slug := slugifyRegexp.ReplaceAllString(name, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func ParseProgram(programBytes util.HexBytes) (*Program, error) {
	var program Program
	if err := json.Unmarshal(programBytes, &program); err != nil {
		return nil, fmt.Errorf("failed to parse JSON program: %w", err)
	}

	if program.Address == "" {
		return nil, fmt.Errorf("missing required field: address")
	}
	return &program, nil
}

func (p *Program) connect(ctx context.Context) (conn net.Conn, err error) {
	conn, err = modbus.Connect(ctx, p.Address)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (p *Program) Run(ctx context.Context) error {
	conn, err := p.connect(ctx)
	if err != nil {
		return err
	}
	defer util.CloseQuietly(conn)

	loops := p.Loops
	if loops <= 0 {
		loops = 1
	}

	util.LogDebug(ctx, "Parsed program", "program", p)

	util.LogDebug(ctx, "Starting command execution",
		"loops", loops)
	for i := 0; i < loops; i++ {
		util.LogDebug(ctx, "Starting loop", "loopNumber", i+1, "loopCount", loops)
		for j, cmdGroup := range p.Commands {
			util.LogDebug(ctx, "Executing command group", "groupNumber", j+1, "group", cmdGroup)
			for k, cmd := range cmdGroup {
				util.LogDebug(ctx, "Executing command", "commandNumber", k+1, "command", cmd)
				modbusMessage, err := cmd.BuildMessage()
				if err != nil {
					return fmt.Errorf("failed to build message in loop %d, command group %d, command %d (%v): %w", i+1, j+1, k+1, cmd, err)
				}
				_, _, err = modbus.Send(ctx, conn, modbusMessage)
				if err != nil {
					return fmt.Errorf("failure in loop %d, command group %d, command %d (%v): %w", i+1, j+1, k+1, cmd, err)
				}
			}

			if p.CommandIntervalMillis > 0 && (j < len(p.Commands)-1 || i < loops-1) {
				util.LogDebug(ctx, "Waiting before next command group", "milliseconds", p.CommandIntervalMillis, "loopNumber", i+1, "commandGroupNumber", j+1)
				time.Sleep(time.Duration(p.CommandIntervalMillis) * time.Millisecond)
			}
		}
	}
	return nil
}
