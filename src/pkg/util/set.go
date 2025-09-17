package util

import "fmt"

type Set struct {
	elements map[string]bool
	count    int
}

func NewSet() *Set {
	return &Set{
		elements: make(map[string]bool),
		count:    0,
	}
}

func (s *Set) Add(element string) {
	value, exists := s.elements[element]
	if exists && value {
		return
	}
	s.elements[element] = true
	s.count++
}

func (s *Set) Remove(element string) {
	value, exists := s.elements[element]
	if exists && value {
		s.count--
		s.elements[element] = false
	}
}

func (s *Set) Contains(element string) bool {
	return s.elements[element]
}

func (s *Set) ToArray() []string {
	fmt.Printf("Set has %d elements: %v\n", s.count, s.elements)
	result := make([]string, s.count)
	for key, exists := range s.elements {
		if exists {
			result = append(result, key)
		}
	}
	return result
}
