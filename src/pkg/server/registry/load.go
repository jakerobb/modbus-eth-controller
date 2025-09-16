package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jakerobb/modbus-eth-controller/pkg/api"
)

func (r *Registry) LoadProgramsFromDir(dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to read program directory %s: %v. No programs will be loaded.\n", dir, err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		fullPath := filepath.Join(dir, file.Name())
		program, err := api.ParseProgramFromFile(fullPath)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to parse program from file %s: %v\n", fullPath, err)
			continue
		}

		_, exists := r.GetProgram(program.Slug)
		if exists {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: duplicate program slug '%s' found in file %s, skipping\n", program.Slug, fullPath)
			continue
		}

		r.StoreProgram(program)

		fmt.Printf("Loaded program '%s' from %s\n", program.Slug, fullPath)
	}
	fmt.Printf("Total programs loaded: %d\n", r.Size())
}

func (r *Registry) LoadNewProgramFromDisk(slug string, dir string) (*api.Program, error) {
	_, exists := r.GetProgram(slug)
	if exists {
		return nil, fmt.Errorf("program '%s' already exists", slug)
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read program directory %s: %v", dir, err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") || api.SlugifyFilename(file.Name()) != slug {
			continue
		}
		fullPath := filepath.Join(dir, file.Name())
		program, err := api.ParseProgramFromFile(fullPath)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to parse program from file %s: %v\n", fullPath, err)
			continue
		}

		_, exists := r.GetProgram(program.Slug)
		if exists {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: duplicate program slug '%s' found in file %s, skipping\n", program.Slug, fullPath)
			continue
		}

		fmt.Printf("Loaded new program '%s' from %s\n", program.Slug, fullPath)
		r.StoreProgram(program)
		return program, nil
	}
	return nil, nil
}

func (r *Registry) ReloadProgramFromDiskIfNewer(program *api.Program) (*api.Program, error) {
	path := program.Path
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat program file: %v", err)
	}
	modTime := info.ModTime()
	if modTime.After(*program.LastModified) {
		newProgram, err := api.ParseProgramFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to reload program: %v", err)
		}
		oldMod := program.LastModified
		r.StoreProgram(newProgram)
		fmt.Printf("Reloaded program '%s': old mod time %v, new mod time %v\n", program.Slug, oldMod, newProgram.LastModified)
		return newProgram, nil
	}
	return program, nil
}
