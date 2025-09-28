// @title           Modbus ETH Controller API
// @version         1.0
// @description     API for controlling and querying Modbus Ethernet relay devices

// @contact.name   Jake Robb
// @contact.url    https://github.com/jakerobb
// @contact.email  jakerobb@gmail.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/jakerobb/modbus-eth-controller/pkg/api"
	"github.com/jakerobb/modbus-eth-controller/pkg/server"
	"github.com/jakerobb/modbus-eth-controller/pkg/util"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	for _, arg := range os.Args[1:] {
		if arg == "--help" {
			printUsage()
			return
		}
		if arg == "--server" {
			s := server.InitServer()
			s.Start()
			return
		}
	}

	slog.SetDefault(logger.With("component", "cli"))
	programs := readInputPrograms()

	if len(programs) == 0 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()

	var err error
	for _, program := range programs {
		programCtx := ctx
		if program.Debug {
			programCtx = context.WithValue(ctx, "debug", program.Debug)
		}
		err = program.Run(programCtx)
		if err != nil {
			slog.Error("Execution of program failed", "path", program.Path, "error", err)
		}
	}
	if err != nil {
		os.Exit(1)
	}
}

func readInputPrograms() []*api.Program {
	var programBytes util.HexBytes
	info, err := os.Stdin.Stat()
	if err != nil {
		slog.Error("Failed to stat stdin", "error", err)
		os.Exit(1)
	}

	programs := make([]*api.Program, 0)

	// Only read stdin if it's being piped or redirected and there is at least one byte available
	if (info.Mode()&os.ModeCharDevice) == 0 && info.Size() > 0 {
		programBytes, err := io.ReadAll(os.Stdin)
		if err == nil {
			program, err := api.ParseProgram(programBytes)
			if err != nil {
				slog.Error("Failed to parse program from stdin", "error", err)
				os.Exit(1)
			}
			program.Path = "[stdin]"
			programs = append(programs, program)
		} else if err.Error() != "EOF" {
			slog.Error("Failed to read program from stdin", "error", err)
		}
	}

	for i, filename := range os.Args[1:] {
		programBytes, err = os.ReadFile(filename)
		if err != nil {
			slog.Error("Failed to read program file", "argIndex", i, "file", filename, "error", err)
		} else {
			program, err := api.ParseProgram(programBytes)
			if err != nil {
				slog.Error("Failed to parse program from file", "argIndex", i, "file", filename, "error", err)
				os.Exit(1)
			}
			program.Path = filename
			programs = append(programs, program)
		}
	}
	return programs
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  modbus-eth-controller --server")
	fmt.Println("      Start the server mode.")
	fmt.Println("  modbus-eth-controller --help")
	fmt.Println("      Show this help message.")
	fmt.Println("  modbus-eth-controller < input.json")
	fmt.Println("      Provide JSON input via stdin.")
	fmt.Println("  modbus-eth-controller file1.json [file2.json ...]")
	fmt.Println("      Provide JSON input via one or more file paths.")
}
