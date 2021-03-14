package goss

// Styles is a collection of Styles
type Styles map[string]Style

// Add adds styles and owewrite the present ones
func (s Styles) Add(o Styles) {
	for k, v := range o {
		if val, ok := s[k]; ok {
			v.Overwrite(val)
		} else {
			s[k] = v
		}
	}
}

// Style is a parsed form of goss syntax
type Style map[string][]interface{}

// Ident returns first string under the property
func (s Style) Ident(key string) (string, bool) {
	val, ok := s[key]
	if !ok {
		return "", false
	}
	v, ok := val[0].(string)
	return v, ok
}

// Int returns first integer under the property
func (s Style) Int(key string) (int, bool) {
	val, ok := s[key]
	if !ok {
		return 0, false
	}
	v, ok := val[0].(int)
	return v, ok
}

// Float returns first float under the property
func (s Style) Float(key string) (float64, bool) {
	val, ok := s[key]
	if !ok {
		return 0, false
	}
	v, ok := val[0].(float64)
	return v, ok
}

// Uint returns first unsigned integer under the property
func (s Style) Uint(key string) (uint64, bool) {
	val, ok := s[key]
	if !ok {
		return 0, false
	}
	v, ok := val[0].(uint64)
	return v, ok
}

// Overwrite overwrites o by s, props can be overwritten and also added
func (s Style) Overwrite(o Style) {
	for k, v := range s {
		nv := make([]interface{}, len(v))
		copy(nv, v)
		o[k] = nv
	}
}

// Inherit makes as inherit all props that are at the same position, if
// s kay contains only one element == "inherit" the whole property of o is inherited
func (s Style) Inherit(o Style) {
	for k, v := range s {
		ov, ok := o[k]
		if !ok {
			continue
		}
		min := min(len(v), len(ov))
		for i := 0; i < min; i++ {
			val, ok := v[i].(string)
			if ok && val == "inherit" {
				if min == 1 && len(v) == 1 {
					cp := make([]interface{}, len(ov))
					copy(cp, ov)
					s[k] = cp
					break
				}
				v[i] = ov[i]
			}
		}
	}
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}
