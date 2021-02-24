package goml

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/jakubDoka/sterr"
)

func TestParse(t *testing.T) {
	p := NParser()
	p.AddDefinitions("div", "fiv", "giv")
	testCases := []struct {
		desc   string
		input  string
		output []Div
		err    sterr.Err
	}{
		{
			desc: "simple",
			input: `
<div> 
	<fiv> 
		<giv/>
		<giv/>
	</>
</>
			`,
			output: []Div{
				{
					Name:       "div",
					Attributes: Attribs{},
					Children: []Div{
						{
							Name:       "fiv",
							Attributes: Attribs{},
							Children: []Div{
								{
									Name:       "giv",
									Attributes: Attribs{},
								},
								{
									Name:       "giv",
									Attributes: Attribs{},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:  "incomplete",
			input: `<`,
			err:   ErrDiv.Incomplete,
		},
		{
			desc:  "after slash",
			input: `<div></`,
			err:   ErrDiv.AfterSlash,
		},
		{
			desc:  "extra closure",
			input: `<div></></>`,
			err:   ErrDiv.ExtraClosure,
		},
		{
			desc:  "unexpected",
			input: `<div>a</></>`,
			err:   ErrByte,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			div, err := p.Parse([]byte(tC.input))
			if !tC.err.SameSurface(err) {
				t.Error(p.err)
				t.Error(string(p.ch), p.stack)
				return
			}

			if p.Failed() {
				return
			}

			if !reflect.DeepEqual(div.Children, tC.output) {
				t.Error(div)
			}
		})
	}
}

func TestDiv(t *testing.T) {
	p := NParser()
	p.AddDefinitions("niv")
	p.ClearDefinitions()
	p.AddDefinitions("div", "fiv", "giv", "riv")
	p.RemoveDefinitions("riv")

	testCases := []struct {
		desc   string
		input  string
		output []Div
		err    sterr.Err
	}{
		{
			desc:  "simple",
			input: `<div hello="hello" krr=["asd" "asd"]/>`,
			output: []Div{
				{
					Name: "div",
					Attributes: Attribs{
						"hello": {"hello"},
						"krr":   {"asd", "asd"},
					},
				},
			},
		},
		{
			desc:  "unfinished",
			input: `<div>`,
			output: []Div{
				{
					Name:       "div",
					Attributes: Attribs{},
				},
			},
		},
		{
			desc:  "missing identifier",
			input: `< div/>`,
			err:   ErrDiv.Identifier,
		},
		{
			desc:  "unknown identifier",
			input: `<riv/>`,
			err:   ErrDiv.Unknown,
		},
		{
			desc:  "invalid end",
			input: `<div/ >`,
			err:   ErrDiv.AfterSlash,
		},
		{
			desc:  "after identifier",
			input: `<div=/>`,
			err:   ErrDiv.AfterIdent,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			p.Restart([]byte(tC.input))
			p.Advance()
			p.Div()
			if !tC.err.SameSurface(p.err) {
				t.Error(p.err)
				return
			}

			if p.Failed() {
				return
			}

			if !reflect.DeepEqual(p.root.Children, tC.output) && !reflect.DeepEqual([]Div(p.stack), tC.output) {
				t.Error(p.root.Children, p.stack, tC.output)
			}
		})
	}
}

func TestParseValue(t *testing.T) {
	p := Parser{}
	testCases := []struct {
		desc   string
		input  string
		output Attribs
		err    sterr.Err
	}{
		{
			desc:  "simple",
			input: `hello="hello"`,
			output: Attribs{
				"hello": {"hello"},
			},
		},
		{
			desc:  "no value",
			input: `hello `,
			output: Attribs{
				"hello": {"true"},
			},
		},
		{
			desc:  "invalid sign",
			input: `hello/`,
			err:   ErrAttrib.Assignmant,
		},
		{
			desc:  "incomplete",
			input: `hello=`,
			err:   ErrAttrib.Incomplete,
		},
		{
			desc:  "invalid start",
			input: `hello= `,
			err:   ErrAttrib.ValueStart,
		},
		{
			desc:  "invalid string",
			input: `hello="br\xfk"`,
			err:   ErrEscape.Illegal,
		},
		{
			desc:  "extra space",
			input: `hello=[ ]`,
			err:   ErrAttrib.ExtraSpace,
		},
		{
			desc:  "incomplete list",
			input: `hello=[`,
			err:   ErrAttrib.ListIncomplete,
		},
		{
			desc:  "invalid byte",
			input: `hello=[x]`,
			err:   ErrAttrib.BetweenByte,
		},
		{
			desc:  "invalid byte",
			input: `hello=[""x]`,
			err:   ErrAttrib.BetweenByte,
		},
		{
			desc:  "list",
			input: `hello=["hello"]`,
			output: Attribs{
				"hello": {"hello"},
			},
		},
		{
			desc:  "long list",
			input: `hello=["hello" "fl" "gg" "mm"]`,
			output: Attribs{
				"hello": {"hello", "fl", "gg", "mm"},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			p.Restart([]byte(tC.input))
			p.parsed = NDiv()
			p.Attribute()
			if !tC.err.SameSurface(p.err) {
				t.Error(p.err)
				t.Error(string(p.ch))
				return
			}

			if p.Failed() {
				return
			}

			cmp(t, p.parsed.Attributes, tC.output)
		})
	}
}

func cmp(t *testing.T, a, b Attribs) {
	if len(a) != len(b) {
		t.Error("len", a)
		return
	}

	for k, v := range a {
		val, ok := b[k]
		if !ok {
			t.Error("key", a)
			return
		}
		if len(v) != len(val) {
			t.Error("inner len", a)
			return
		}
		for i, v := range v {
			if v != val[i] {
				t.Error("element", a)
				return
			}
		}
	}
}

func TestParseString(t *testing.T) {
	p := Parser{}

	testCases := []struct {
		desc          string
		output, input string
		err           sterr.Err
	}{
		{
			desc:   "simple",
			input:  "hello there\"",
			output: "hello there",
		},
		{
			desc:   "runeSelf",
			input:  "они\"",
			output: "они",
		},
		{
			desc:  "runeSelf fail",
			input: "\xF0\"",
			err:   ErrInvalidRune,
		},
		{
			desc:  "not terminated",
			input: "asd",
			err:   ErrStringNotTerminated,
		},
		{
			desc:  "escape not terminated",
			input: "\\",
			err:   ErrEscape.Incomplete,
		},
		{
			desc:   "simple escape",
			input:  "\\a\\b\\v\\n\\r\\t\\a\\f\\\\\\\"\"",
			output: "\a\b\v\n\r\t\a\f\\\"",
		},
		{
			desc:   "octal",
			input:  "\\123\"",
			output: "\123",
		},
		{
			desc:  "octal not terminated",
			input: "\\12",
			err:   ErrEscape.Incomplete,
		},
		{
			desc:  "octal invalid character",
			input: "\\128\"",
			err:   ErrEscape.Illegal,
		},
		{
			desc:  "octal overflow",
			input: "\\777\"",
			err:   ErrEscape.Overflow,
		},
		{
			desc:   "x parsing",
			input:  "\\xFF\"",
			output: "ÿ",
		},
		{
			desc:  "x parsing not terminated",
			input: "\\xF",
			err:   ErrEscape.Incomplete,
		},
		{
			desc:  "x parsing invalid byte",
			input: "\\xFX\"",
			err:   ErrEscape.Illegal,
		},
		{
			desc:   "u parsing",
			input:  "\\uff00\"",
			output: "\uFF00",
		},
		{
			desc:   "U parsing",
			input:  "\\U000000FF\"",
			output: "\U000000FF",
		},
		{
			desc:  "U parsing overflow",
			input: "\\UFFFFFFFF\"",
			err:   ErrEscape.Overflow,
		},
		{
			desc:  "invalid escape ident",
			input: "\\kFF\"",
			err:   ErrEscape.InvalidIdent,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			p.Restart([]byte(tC.input))
			p.String()
			fmt.Println(tC.input)
			if !tC.err.SameSurface(p.err) {
				t.Error(p.err)
				return
			}

			if p.Failed() {
				return
			}

			res := string(p.stringBuff)
			if res != tC.output {
				t.Errorf("%q != %q || %v != %v", res, tC.output, res, tC.output)
			}
		})
	}
}
