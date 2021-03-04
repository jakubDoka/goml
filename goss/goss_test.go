package goss

import (
	"testing"

	"github.com/jakubDoka/goml/core"
	"github.com/jakubDoka/sterr"
)

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
			input: `a:+;`,
			err:   ErrIdent,
		},
		{
			desc:  "no ':'",
			input: `a:b;`,
			err:   ErrExpectedByte,
		},
		{
			desc:  "no value",
			input: `a:b:  ;`,
			err:   ErrExpectedValue,
		},
		{
			desc: "all features",
			input: `a:
b: 10i;
c: 11f;
e: hello;
d: kl ml f 10u 10;
			;`,
			out: Styles{
				"a": {
					"b": {10},
					"c": {float64(11)},
					"e": {"hello"},
					"d": {"kl", "ml", "f", uint64(10), 10},
				},
			},
		},
		{
			desc:  "all features one line",
			input: `a:b: 10i;c: 11f;e: hello;d: kl ml f 10u 10;;`,
			out: Styles{
				"a": {
					"b": {10},
					"c": {float64(11)},
					"e": {"hello"},
					"d": {"kl", "ml", "f", uint64(10), 10},
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			out, err := p.Parse([]byte(tC.input))
			if !tC.err.SameSurface(err) {
				t.Error(err)
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
		"d": {"kl", "ml", "f", uint64(10), 10},
	}

	s, err := p.Style([]byte("b: 10i;c: 11f;e: hello;d: kl ml f 10u 10;"))
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
