package registry

import (
	"sync"

	"github.com/jakerobb/modbus-eth-controller/pkg/api"
)

type Registry struct {
	Programs map[string]*api.Program `json:"status"`
	mutex    sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		Programs: make(map[string]*api.Program),
		mutex:    sync.RWMutex{},
	}
}

func (r *Registry) GetProgram(slug string) (*api.Program, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	program, exists := r.Programs[slug]
	return program, exists
}

func (r *Registry) StoreProgram(program *api.Program) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.Programs[program.Slug] = program
}

func (r *Registry) Size() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.Programs)
}
