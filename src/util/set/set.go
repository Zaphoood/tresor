package set

type Set[T comparable] struct {
	_map map[T]any
}

func New[T comparable]() Set[T] {
	return Set[T]{map[T]any{}}
}

func (s *Set[T]) Insert(val T) {
	s._map[val] = struct{}{}
}

func (s *Set[T]) Delete(val T) {
	delete(s._map, val)
}

func (s Set[T]) Contains(val T) bool {
	_, contained := s._map[val]
	return contained
}
