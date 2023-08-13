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

func (m *Map[T]) Delete(id int) {
	m.sm.Delete(id)
}

func (m *Map[T]) Range(f func(id int, t T) bool) {
	m.sm.Range(
		func(key any, value any) bool {
			return f(key.(int), value.(T))
		})
}
