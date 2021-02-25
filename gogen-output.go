package goml

// DivStack ...
type DivStack []Element

// Top returns to element of stack
func (s DivStack) Top() *Element {
	return &s[len(s)-1]
}

// Push appends the value
func (s *DivStack) Push(v Element) {
	*s = append(*s, v)
}

// Pop pos an element but does not take in to account the memory leak
func (s *DivStack) Pop() Element {
	sv := *s
	l := len(sv) - 1
	val := sv[l]
	*s = sv[:l]
	return val
}

// CanPop returns whether you can use Pop without out of bounds panic
func (s DivStack) CanPop() bool {
	return len(s) != 0
}
