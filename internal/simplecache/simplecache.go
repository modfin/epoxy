package simplecache

import (
	"sync"
	"time"
)

type Cache interface {
	Set(key string, value string)
	Get(key string) string
}

func New(swapInterval time.Duration) Cache {
	s := &simpleCache{}
	s.swapInterval = swapInterval
	s.buffers = make([]map[string]string, 2)
	s.buffers[0] = make(map[string]string)
	s.buffers[1] = make(map[string]string)
	return s
}

type simpleCache struct {
	mu           sync.RWMutex
	swapInterval time.Duration
	buffers      []map[string]string
	lastSwap     time.Time
}

func (s *simpleCache) Set(key string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Now().Sub(s.lastSwap) > s.swapInterval {
		s.buffers[1] = s.buffers[0]
		s.buffers[0] = make(map[string]string)
		s.lastSwap = time.Now()
	}
	s.buffers[0][key] = value
}

func (s *simpleCache) Get(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.buffers == nil {
		return ""
	}
	if v, ok := s.buffers[0][key]; ok {
		return v
	}
	v := s.buffers[1][key]
	if v != "" {
		go func() {
			s.Set(key, v)
		}()
	}
	return v
}
