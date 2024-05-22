/*
Open Source Initiative OSI - The MIT License (MIT):Licensing

The MIT License (MIT)
Copyright (c) 2013 - 2022 Ralph Caraveo (deckarep@gmail.com)

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package mapset

import "sync"

type threadSafeSet[T comparable] struct {
	sync.RWMutex
	uss threadUnsafeSet[T]
}

func newThreadSafeSet[T comparable]() threadSafeSet[T] {
	newUss := newThreadUnsafeSet[T]()
	return threadSafeSet[T]{
		uss: newUss,
	}
}

func (s *threadSafeSet[T]) Add(v T) bool {
	s.Lock()
	ret := s.uss.Add(v)
	s.Unlock()
	return ret
}

func (s *threadSafeSet[T]) Contains(v ...T) bool {
	s.RLock()
	ret := s.uss.Contains(v...)
	s.RUnlock()
	return ret
}

func (s *threadSafeSet[T]) IsSubset(other Set[T]) bool {
	o := other.(*threadSafeSet[T])

	s.RLock()
	o.RLock()

	ret := s.uss.IsSubset(&o.uss)
	s.RUnlock()
	o.RUnlock()
	return ret
}

func (s *threadSafeSet[T]) IsProperSubset(other Set[T]) bool {
	o := other.(*threadSafeSet[T])

	s.RLock()
	defer s.RUnlock()
	o.RLock()
	defer o.RUnlock()

	return s.uss.IsProperSubset(&o.uss)
}

func (s *threadSafeSet[T]) IsSuperset(other Set[T]) bool {
	return other.IsSubset(s)
}

func (s *threadSafeSet[T]) IsProperSuperset(other Set[T]) bool {
	return other.IsProperSubset(s)
}

func (s *threadSafeSet[T]) Union(other Set[T]) Set[T] {
	o := other.(*threadSafeSet[T])

	s.RLock()
	o.RLock()

	unsafeUnion := s.uss.Union(&o.uss).(*threadUnsafeSet[T])
	ret := &threadSafeSet[T]{uss: *unsafeUnion}
	s.RUnlock()
	o.RUnlock()
	return ret
}

func (s *threadSafeSet[T]) Intersect(other Set[T]) Set[T] {
	o := other.(*threadSafeSet[T])

	s.RLock()
	o.RLock()

	unsafeIntersection := s.uss.Intersect(&o.uss).(*threadUnsafeSet[T])
	ret := &threadSafeSet[T]{uss: *unsafeIntersection}
	s.RUnlock()
	o.RUnlock()
	return ret
}

func (s *threadSafeSet[T]) Difference(other Set[T]) Set[T] {
	o := other.(*threadSafeSet[T])

	s.RLock()
	o.RLock()

	unsafeDifference := s.uss.Difference(&o.uss).(*threadUnsafeSet[T])
	ret := &threadSafeSet[T]{uss: *unsafeDifference}
	s.RUnlock()
	o.RUnlock()
	return ret
}

func (s *threadSafeSet[T]) SymmetricDifference(other Set[T]) Set[T] {
	o := other.(*threadSafeSet[T])

	s.RLock()
	o.RLock()

	unsafeDifference := s.uss.SymmetricDifference(&o.uss).(*threadUnsafeSet[T])
	ret := &threadSafeSet[T]{uss: *unsafeDifference}
	s.RUnlock()
	o.RUnlock()
	return ret
}

func (s *threadSafeSet[T]) Clear() {
	s.Lock()
	s.uss = newThreadUnsafeSet[T]()
	s.Unlock()
}

func (s *threadSafeSet[T]) Remove(v T) {
	s.Lock()
	delete(s.uss, v)
	s.Unlock()
}

func (s *threadSafeSet[T]) Cardinality() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.uss)
}

func (s *threadSafeSet[T]) Each(cb func(T) bool) {
	s.RLock()
	for elem := range s.uss {
		if cb(elem) {
			break
		}
	}
	s.RUnlock()
}

func (s *threadSafeSet[T]) Iter() <-chan T {
	ch := make(chan T)
	go func() {
		s.RLock()

		for elem := range s.uss {
			ch <- elem
		}
		close(ch)
		s.RUnlock()
	}()

	return ch
}

func (s *threadSafeSet[T]) Iterator() *Iterator[T] {
	iterator, ch, stopCh := newIterator[T]()

	go func() {
		s.RLock()
	L:
		for elem := range s.uss {
			select {
			case <-stopCh:
				break L
			case ch <- elem:
			}
		}
		close(ch)
		s.RUnlock()
	}()

	return iterator
}

func (s *threadSafeSet[T]) Equal(other Set[T]) bool {
	o := other.(*threadSafeSet[T])

	s.RLock()
	o.RLock()

	ret := s.uss.Equal(&o.uss)
	s.RUnlock()
	o.RUnlock()
	return ret
}

func (s *threadSafeSet[T]) Clone() Set[T] {
	s.RLock()

	unsafeClone := s.uss.Clone().(*threadUnsafeSet[T])
	ret := &threadSafeSet[T]{uss: *unsafeClone}
	s.RUnlock()
	return ret
}

func (s *threadSafeSet[T]) String() string {
	s.RLock()
	ret := s.uss.String()
	s.RUnlock()
	return ret
}

func (s *threadSafeSet[T]) Pop() (T, bool) {
	s.Lock()
	defer s.Unlock()
	return s.uss.Pop()
}

func (s *threadSafeSet[T]) ToSlice() []T {
	keys := make([]T, 0, s.Cardinality())
	s.RLock()
	for elem := range s.uss {
		keys = append(keys, elem)
	}
	s.RUnlock()
	return keys
}

func (s *threadSafeSet[T]) MarshalJSON() ([]byte, error) {
	s.RLock()
	b, err := s.uss.MarshalJSON()
	s.RUnlock()

	return b, err
}

func (s *threadSafeSet[T]) UnmarshalJSON(p []byte) error {
	s.RLock()
	err := s.uss.UnmarshalJSON(p)
	s.RUnlock()

	return err
}
