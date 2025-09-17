// @title           Modbus ETH Controller API
// @version         1.0
// @description     API for controlling and querying Modbus Ethernet relay devices

// @contact.name   Jake Robb
// @contact.url    https://github.com/jakerobb
// @contact.email  jakerobb@gmail.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/jakerobb/modbus-eth-controller/pkg/api"
	"github.com/jakerobb/modbus-eth-controller/pkg/server"
)

func main() {
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

	var programBytes []byte
	var err error

	info, err := os.Stdin.Stat()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to stat stdin: %v\n", err)
		os.Exit(1)
	}

	programs := readInputPrograms(info, programBytes, err)

	if len(programs) == 0 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()

	for _, program := range programs {
		programCtx := ctx
		if program.Debug {
			programCtx = context.WithValue(ctx, "debug", program.Debug)
		}
		err = program.Run(programCtx)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Execution of program '%s' failed: %v\n", program.Path, err)
		}
	}
}

func readInputPrograms(info os.FileInfo, programBytes []byte, err error) []*api.Program {
	programs := make([]*api.Program, 0)

	// Only read stdin if it's being piped or redirected and there is at least one byte available
	if (info.Mode()&os.ModeCharDevice) == 0 && info.Size() > 0 {
		programBytes, err := io.ReadAll(os.Stdin)
		if err == nil {
			program, err := api.ParseProgram(programBytes)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Failed to parse program from stdin: %v\n", err)
				os.Exit(1)
			}
			program.Path = "[stdin]"
			programs = append(programs, program)
		} else if err.Error() != "EOF" {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to read program: %v\n", err)
		}
	}

	for i, filename := range os.Args[1:] {
		programBytes, err = os.ReadFile(filename)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to read program file %s at argument index %d: %v\n", filename, i, err)
		} else {
			program, err := api.ParseProgram(programBytes)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Failed to parse program from file %s: %v\n", filename, err)
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
