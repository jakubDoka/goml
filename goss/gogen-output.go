package goss


// Stack ...
type Stack []stl

// Top returns to element of stack
func (s Stack) Top() *stl {
	return &s[len(s)-1]
}

// Push appends the value
func (s *Stack) Push(v stl) {
	*s = append(*s, v)
}

// Pop pos an element but does not take in to account the memory leak
func (s *Stack) Pop() stl {
	sv := *s
	l := len(sv) - 1
	val := sv[l]
	*s = sv[:l]
	return val
}

// CanPop returns whether you can use Pop without out of bounds panic
func (s Stack) CanPop() bool {
	return len(s) != 0
}

