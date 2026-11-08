package base

type Item[T any] struct {
	value  T
	left   *Item[T]
	right  *Item[T]
	parent *Item[T]
}

type Heap[T any] struct {
	size int
	peek *Item[T]
	less func(a, b T) bool
}

func NewHeap[T any](less func(a, b T) bool) *Heap[T] {
	return &Heap[T]{less: less}
}

func (h *Heap[T]) Len() int { return h.size }

func (h *Heap[T]) Peek() (T, bool) {
	if h.size == 0 {
		var zero T
		return zero, false
	}
	return h.peek.value, true
}

func (h *Heap[T]) merge(a, b *Item[T]) *Item[T] {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if h.less(b.value, a.value) {
		a, b = b, a // a — с меньшим значением
	}
	r := h.merge(a.left, a.right)
	a.left, a.right = b, r
	b.parent = a
	if r != nil {
		r.parent = a
	}
	return a
}

func (h *Heap[T]) Push(value T) {
	h.peek = h.merge(h.peek, &Item[T]{value: value})
	h.peek.parent = nil
	h.size++
}

func (h *Heap[T]) Pop() (T, bool) {
	if h.size == 0 {
		var zero T
		return zero, false
	}
	top := h.peek.value
	h.peek = h.merge(h.peek.left, h.peek.right)
	if h.peek != nil {
		h.peek.parent = nil
	}
	h.size--
	return top, true
}
