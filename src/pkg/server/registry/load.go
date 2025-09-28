package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jakerobb/modbus-eth-controller/pkg/api"
	"github.com/jakerobb/modbus-eth-controller/pkg/util"
)

func (r *Registry) LoadProgramsFromDir(ctx context.Context, dir string) {
	files, err := os.ReadDir(dir)
	logger := util.GetLogger(ctx)
	if err != nil {
		logger.Error("Failed to read program directory. No programs will be loaded.", "dir", dir, "error", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		fullPath := filepath.Join(dir, file.Name())
		program, err := api.ParseProgramFromFile(fullPath)
		if err != nil {
			logger.Error("Failed to parse program from file. Skipping.", "file", fullPath, "error", err)
			continue
		}

		_, exists := r.GetProgram(program.Slug)
		if exists {
			logger.Warn("Duplicate program slug found. Skipping.", "slug", program.Slug, "file", fullPath)
			continue
		}

		r.StoreProgram(program)

		logger.Info("Loaded program",
			"slug", program.Slug,
			"path", fullPath)
	}
	logger.Info("Loaded programs",
		"programCount", r.Size())
}

func (r *Registry) LoadNewProgramFromDisk(ctx context.Context, slug string, dir string) (*api.Program, error) {
	logger := util.GetLogger(ctx)

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
			logger.Error("Failed to parse program from file. Skipping.", "file", fullPath, "error", err)
			continue
		}

		_, exists := r.GetProgram(program.Slug)
		if exists {
			logger.Warn("Duplicate program slug found. Skipping.", "slug", program.Slug, "file", fullPath)
			continue
		}

		logger.Info("Loaded new program",
			"slug", program.Slug,
			"path", fullPath)
		r.StoreProgram(program)
		return program, nil
	}
	return nil, nil
}

func (r *Registry) ReloadProgramFromDiskIfNewer(ctx context.Context, program *api.Program) (*api.Program, error) {
	logger := util.GetLogger(ctx)
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
		logger.Info("Reloaded program",
			"slug", program.Slug,
			"oldModTime", oldMod,
			"newModTime", newProgram.LastModified)
		return newProgram, nil
	}
	return program, nil
}
