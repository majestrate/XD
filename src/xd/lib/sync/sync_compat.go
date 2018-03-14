// +build !go1.9

package sync

import (
	"sync"
)

type Pool struct {
	sync.Pool
}

type WaitGroup struct {
	sync.WaitGroup
}

type Mutex struct {
	sync.Mutex
}

type RWMutex struct {
	sync.RWMutex
}

type Map struct {
	data   map[interface{}]interface{}
	access Mutex
}

func (m *Map) ensure() {
	if m.data == nil {
		m.data = make(map[interface{}]interface{})
	}
}

func (m *Map) Delete(key interface{}) {
	m.access.Lock()
	m.ensure()
	delete(m.data, key)
	m.access.Unlock()
}

func (m *Map) Load(key interface{}) (actual interface{}, ok bool) {
	m.access.Lock()
	m.ensure()
	actual, ok = m.data[key]
	m.access.Unlock()
	return
}

func (m *Map) LoadOrStore(key, value interface{}) (actual interface{}, loaded bool) {
	m.access.Lock()
	m.ensure()
	actual, loaded = m.data[key]
	if !loaded {
		m.data[key] = value
		actual = value
	}
	m.access.Unlock()
	return
}

func (m *Map) Range(f func(key, value interface{}) bool) {
	mcopy := make(map[interface{}]interface{})
	m.access.Lock()
	m.ensure()
	for k := range m.data {
		mcopy[k] = m.data[k]
	}
	m.access.Unlock()
	for k := range mcopy {
		if !f(k, mcopy[k]) {
			return
		}
	}
}

func (m *Map) Store(key, value interface{}) {
	m.access.Lock()
	m.ensure()
	m.data[key] = value
	m.access.Unlock()
}
