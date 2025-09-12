package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	var programBytes []byte
	var err error

	// Check if --server flag is present
	for _, arg := range os.Args[1:] {
		if arg == "--server" {
			StartServer()
			return
		}
	}

	// Read JSON object from stdin
	info, err := os.Stdin.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stat stdin: %v\n", err)
		os.Exit(1)
	}

	programs := readInputPrograms(info, programBytes, err)

	for _, program := range programs {
		program.Run()
	}
}

func readInputPrograms(info os.FileInfo, programBytes []byte, err error) []*Program {
	programs := make([]*Program, 0)

	// Only read stdin if it's being piped or redirected and there is at least one byte available
	if (info.Mode()&os.ModeCharDevice) == 0 && info.Size() > 0 {
		programBytes, err := io.ReadAll(os.Stdin)
		if err == nil {
			program, err := ParseProgram(programBytes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse program from stdin: %v\n", err)
				os.Exit(1)
			}
			programs = append(programs, program)
		} else if err.Error() != "EOF" {
			fmt.Fprintf(os.Stderr, "Failed to read program: %v\n", err)
		}
	}

	for i, filename := range os.Args[1:] {
		programBytes, err = os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read program file %s at argument index %d: %v\n", filename, i, err)
		} else {
			program, err := ParseProgram(programBytes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse program from file %s: %v\n", filename, err)
				os.Exit(1)
			}
			programs = append(programs, program)
		}
	}
	return programs
}
