package mapcache

import (
	"sync"
)

type Map[T any] struct {
	sm sync.Map
}

func (m *Map[T]) Get(id int) T {
	var t T
	v, _ := m.sm.LoadOrStore(id, t)
	return v.(T)
}

func (m *Map[T]) Set(id int, t T) {
	m.sm.Store(id, t)
}
