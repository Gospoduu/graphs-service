package base

import "sync"

type Stack[T any] struct {
	items []T
	mx    sync.RWMutex
}

func NewStack[T any](capacity int) *Stack[T] {
	return &Stack[T]{
		items: make([]T, 0, capacity),
	}
}

func (s *Stack[T]) Push(value T) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.items = append(s.items, value)
}

func (s *Stack[T]) Pop() (T, bool) {
	s.mx.Lock()
	defer s.mx.Unlock()

	if len(s.items) == 0 {
		var zero T
		return zero, false
	}

	last := len(s.items) - 1
	value := s.items[last]

	var zero T
	s.items[last] = zero
	s.items = s.items[:last]

	return value, true
}

func (s *Stack[T]) Len() int {
	s.mx.RLock()
	defer s.mx.RUnlock()

	return len(s.items)
}

func (s *Stack[T]) Peek() (T, bool) {
	s.mx.RLock()
	defer s.mx.RUnlock()

	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	return s.items[len(s.items)-1], true
}
