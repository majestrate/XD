// +build go1.9

package sync

import (
	"sync"
)

type Mutex = sync.Mutex
type RWMutex = sync.RWMutex
type WaitGroup = sync.WaitGroup
type Map = sync.Map
type Pool = sync.Pool
