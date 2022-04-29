package proxyd

import "sync"

type StringSet struct {
	underlying map[string]bool
	mtx        sync.RWMutex
}

func NewStringSet() *StringSet {
	return &StringSet{
		underlying: make(map[string]bool),
	}
}

func NewStringSetFromStrings(in []string) *StringSet {
	underlying := make(map[string]bool)
	for _, str := range in {
		underlying[str] = true
	}
	return &StringSet{
		underlying: underlying,
	}
}

func (s *StringSet) Has(test string) bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.underlying[test]
}

func (s *StringSet) Add(str string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.underlying[str] = true
}

func (s *StringSet) Entries() []string {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	out := make([]string, len(s.underlying))
	var i int
	for entry := range s.underlying {
		out[i] = entry
		i++
	}
	return out
}

func (s *StringSet) Extend(in []string) *StringSet {
	out := NewStringSetFromStrings(in)
	for k := range s.underlying {
		out.Add(k)
	}
	return out
}
