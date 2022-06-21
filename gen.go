package anyx

type mapper[K comparable, V any] map[K]V

func (m mapper[K, V]) Len() int {
	return len(m)
}

func (m *mapper[K, V]) Set(k any, v any) {
	if *m == nil {
		*m = map[K]V{}
	}
	(*m)[k.(K)] = v.(V)
}

func (m mapper[K, V]) Get(k any) any {
	return m[k.(K)]
}

func (m mapper[K, V]) Has(k any) bool {
	_, ok := m[k.(K)]
	return ok
}

func (m mapper[K, V]) Keys() (out []Value) {
	out = make([]Value, 0, len(m))
	for k := range m {
		out = append(out, Value{k})
	}
	return
}

func (m mapper[K, V]) Values() (out []Value) {
	out = make([]Value, 0, len(m))
	for _, v := range m {
		out = append(out, Value{v})
	}
	return
}

func (m mapper[K, V]) ForEach(fn func(key any, value Value) (cnt bool)) {
	for k, v := range m {
		if !fn(k, Value{v}) {
			break
		}
	}
}

func MapOf[K comparable, V any, M ~map[K]V](m M) Value {
	return Value{v: mapper[K, V](m)}
}

type slicer[E any] []E

func (s slicer[E]) Len() int {
	return len(s)
}

func (s *slicer[E]) Append(v any) {
	*s = append(*s, v.(E))
}

func (s *slicer[E]) SetAt(k int, v any) {
	(*s)[k] = v.(E)
}

func (s slicer[E]) Get(k any) any {
	return s[k.(int)]
}

func (s slicer[E]) Has(k any) bool {
	for _, v := range s {
		if any(v) == k {
			return true
		}
	}
	return false
}

func (s slicer[E]) Values() (out []Value) {
	out = make([]Value, 0, len(s))
	for _, v := range s {
		out = append(out, Value{v})
	}
	return
}

func (s slicer[E]) ForEach(fn func(key any, value Value) (cnt bool)) {
	for i, v := range s {
		if !fn(i, Value{v}) {
			break
		}
	}
}

func SliceOf[E any, S ~[]E](s S) Value {
	return Value{v: slicer[E](s)}
}
