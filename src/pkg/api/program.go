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

	. "github.com/jakerobb/modbus-eth-controller/pkg"
	"github.com/jakerobb/modbus-eth-controller/pkg/modbus"
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
	defer CloseQuietly(conn)

	loops := p.Loops
	if loops <= 0 {
		loops = 1
	}

	LogDebug(ctx, "Parsed program: %+v\n", p)

	LogDebug(ctx, "Starting command execution for %d loops\n", loops)
	for i := 0; i < loops; i++ {
		LogDebug(ctx, "Loop %d/%d\n", i+1, loops)
		for j, cmdGroup := range p.Commands {
			LogDebug(ctx, "  Executing command group %d: %+v\n", j+1, cmdGroup)
			for k, cmd := range cmdGroup {
				LogDebug(ctx, "    Executing command %d: %+v\n", k+1, cmd)
				modbusMessage := cmd.BuildMessage()
				_, _, err := modbus.Send(ctx, conn, modbusMessage)
				if err != nil {
					return fmt.Errorf("failure in loop %d, command group %d, command %d (%v): %w", i, j, k, cmd, err)
				}
			}

			if p.CommandIntervalMillis > 0 {
				LogDebug(ctx, "  Waiting for %d milliseconds before next command group\n", p.CommandIntervalMillis)
				time.Sleep(time.Duration(p.CommandIntervalMillis) * time.Millisecond)
			}
		}
	}
	return nil
}
