package goss

import (
	"testing"

	"github.com/jakubDoka/goml/core"
	"github.com/jakubDoka/sterr"
)

func TestReadme(t *testing.T) {
	p := Parser{}
	stl, err := p.Parse([]byte(`
style{
    some_floats: 10f 10.4f;
    some_integers: 1i -1i;
    some_strings: hello slack nice;
    everything_together: hello 10i 4.4f -2i 4 1000000;
    sub_style{
        anonymous: {a:b;c:d;} {e:f;i:j;};
    }
}
another_style{
    property: value;
}
	`))
	if err != nil {
		panic(err)
	}
	res := Styles{
		"another_style": {
			"property": {"value"},
		},
		"style": {
			"everything_together": {"hello", 10, 4.4, -2, 4, 1000000},
			"some_floats":         {float64(10), 10.4},
			"some_integers":       {1, -1},
			"some_strings":        {"hello", "slack", "nice"},
			"sub_style": {Style{
				"anonymous": {Style{"a": {"b"}, "c": {"d"}}, Style{"e": {"f"}, "i": {"j"}}},
			}},
		},
	}

	core.TestEqual(t, stl, res)
}

func TestParse(t *testing.T) {
	p := Parser{}
	testCases := []struct {
		desc  string
		input string
		out   Styles
		err   sterr.Err
	}{
		{
			desc:  "no ident",
			input: `a{+}`,
			err:   ErrIdent,
		},
		{
			desc:  "no ':'",
			input: `a{b}`,
			err:   ErrExpectedByte,
		},
		{
			desc:  "no value",
			input: `a{b: a}`,
			err:   ErrExpectedValue,
		},
		{
			desc: "all features",
			input: `
a{
	b: 10i;
	c: 11f;
	e: hello;
	d: kl ml f 10 {
		h: 10;
		k: 3;
		s: hello ml kl;
	};
			}`,
			out: Styles{
				"a": {
					"b": {10},
					"c": {float64(11)},
					"e": {"hello"},
					"d": {"kl", "ml", "f", 10, Style{
						"h": {10},
						"k": {3},
						"s": {"hello", "ml", "kl"},
					}},
				},
			},
		},
		{
			desc:  "all features one line",
			input: `a{b: 10i;c: 11f;e: hello;d: kl ml f 10;f{a:10;b:2;k:h k j;}}`,
			out: Styles{
				"a": {
					"b": {10},
					"c": {float64(11)},
					"e": {"hello"},
					"d": {"kl", "ml", "f", 10},
					"f": {Style{
						"a": {10},
						"b": {2},
						"k": {"h", "k", "j"},
					}},
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			out, err := p.Parse([]byte(tC.input))
			if !tC.err.SameSurface(err) {
				t.Error(err)
				t.Error(sterr.ReadTrace(err))
			}

			if p.Failed() {
				return
			}

			core.TestEqual(t, out, tC.out)
		})
	}
}

func TestParserStyle(t *testing.T) {
	p := Parser{}

	st := Style{
		"b": {10},
		"c": {float64(11)},
		"e": {"hello"},
		"d": {"kl", "ml", "f", 10},
	}

	s, err := p.Style([]byte("b: 10i;c: 11f;e: hello;d: kl ml f 10;"))
	if err != nil {
		t.Error(err)
		return
	}

	core.TestEqual(t, s, st)
}

func TestStyleInherit(t *testing.T) {
	a := Style{
		"a": {"inherit"},
		"b": {"inherit", 10},
		"c": {10, 10, "inherit"},
		"d": {"inherit"},
	}
	b := Style{
		"a": {"a", "b", "c"},
		"b": {100},
		"c": {10, 10, 20},
	}
	res := Style{
		"a": {"a", "b", "c"},
		"b": {100, 10},
		"c": {10, 10, 20},
		"d": {"inherit"},
	}

	a.Inherit(b)

	core.TestEqual(t, a, res)
}

func TestStyle(t *testing.T) {
	s := Style{
		"a": {10},
		"b": {"hello"},
		"c": {20.2},
		"d": {uint64(10)},
	}

	v1, ok := s.Int("a")
	if !ok || v1 != 10 {
		t.Error("int")
	}

	v2, ok := s.Ident("b")
	if !ok || v2 != "hello" {
		t.Error("ident")
	}

	v3, ok := s.Float("c")
	if !ok || v3 != 20.2 {
		t.Error("float")
	}

	v4, ok := s.Uint("d")
	if !ok || v4 != 10 {
		t.Error("uint")
	}

	v1, ok = s.Int("f")
	if ok {
		t.Error("no int")
	}

	v2, ok = s.Ident("f")
	if ok {
		t.Error("no ident")
	}

	v3, ok = s.Float("f")
	if ok {
		t.Error("no float")
	}

	v4, ok = s.Uint("f")
	if ok {
		t.Error("no uint")
	}
}
