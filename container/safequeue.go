package container

import "sync"

type SafeQueue[T any] struct {
	items []T
	head  int
	lock  sync.Mutex
}

func (s *SafeQueue[T]) Push(item T) {
	s.lock.Lock()
	s.items = append(s.items, item)
	s.lock.Unlock()
}

func (s *SafeQueue[T]) Pop() (T, bool) {
	var zero T
	s.lock.Lock()

	defer s.lock.Unlock()

	if s.head >= len(s.items) {
		return zero, false
	}

	v := s.items[s.head]
	s.head++

	// 메모리 회수: head가 전체 길이의 절반 이상이면 슬라이스 축소
	if s.head*2 >= len(s.items) {
		s.items = s.items[s.head:]
		s.head = 0
	}

	return v, true
}

func (s *SafeQueue[T]) Len() int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return len(s.items) - s.head
}
