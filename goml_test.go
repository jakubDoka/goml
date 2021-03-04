package goml

import (
	"reflect"
	"testing"

	"github.com/jakubDoka/goml/core"
	"github.com/jakubDoka/goml/goss"
	"github.com/jakubDoka/sterr"
)

func Test(t *testing.T) {
	gp := goss.Parser{}
	p := NParser(&gp)
	p.AddDefinitions("div")
	elem, err := p.Parse([]byte(`<div style="a: f;k: 10;h: 10f;"/>`))
	if err != nil {
		t.Error(err)
		return
	}

	res := []Element{
		{
			Name: "div",
			Attributes: Attribs{
				"style": {"a: f;k: 10;h: 10f;"},
			},
			Style: goss.Style{
				"a": {"f"},
				"k": {int(10)},
				"h": {float64(10)},
			},
		},
	}

	core.TestEqual(t, res, elem.Children)

	elem, err = p.Parse([]byte(`<div style="a:f;k: 10;h: 10f;"/>`))
	if err == nil {
		t.Errorf("should fail")
	}
}

type pr = map[string]Element

func TestShowcase(t *testing.T) {
	p := NParser(nil)
	p.AddDefinitions("div")
	d, err := p.Parse([]byte(`
<#>prefab definition<#>
<!yes_no>
    <div> 
        <button onclick={yes}>yes</>
        <button onclick={no}>no</>
    </>
<!/>

<div>Hello, is monday today?</>
<yes_no yes="yes-handler-link" no="no-handler-link"/>
	`))

	_ = d

	if err != nil {
		t.Error(err)
		return
	}

	/*b, err := json.MarshalIndent(d, "", "\t")
	if err != nil {
		t.Error(err)
		return
	}
	t.Error(string(b))*/
}

func TestPrefabGeneration(t *testing.T) {
	p := NParser(nil)
	p.AddDefinitions("div")
	testCases := []struct {
		desc   string
		input  string
		output []Element
		err    sterr.Err
	}{
		{
			desc: "simple",
			input: `
			<!h> 
				<div/>
			<!/>

			<h/>
			`,
			output: []Element{
				{
					Name:       "div",
					Attributes: Attribs{},
					Children:   []Element{},
				},
			},
		},
		{
			desc: "with attrib",
			input: `
			<!h> 
				<div h={h}/>
			<!/>

			<h h="h"/>
			`,
			output: []Element{
				{
					Name: "div",
					Attributes: Attribs{
						"h": {"h"},
					},
					Children: []Element{},
					prefabData: []prefabData{
						{
							Name:   "h",
							Target: "h",
							Idx:    -1,
						},
					},
				},
			},
		},
		{
			desc: "with list",
			input: `
			<!h> 
				<div h=[{h} {k} {j}]/>
			<!/>

			<h h="h" k="k"/>
			`,
			output: []Element{
				{
					Name: "div",
					Attributes: Attribs{
						"h": {"h", "k", ""},
					},
					Children: []Element{},
					prefabData: []prefabData{
						{
							Name:   "h",
							Target: "h",
							Idx:    0,
						},
						{
							Name:   "k",
							Target: "h",
							Idx:    1,
						},
						{
							Name:   "j",
							Target: "h",
							Idx:    2,
						},
					},
				},
			},
		},
		{
			desc: "with list",
			input: `
			<!h> 
				<div h="hello {there}"/>
			<!/>

			<h there="meme"/>
			`,
			output: []Element{
				{
					Name: "div",
					Attributes: Attribs{
						"h": {"hello meme"},
					},
					Children: []Element{},
					prefabData: []prefabData{
						{
							Name:   "there",
							Target: "h",
							Idx:    -2,
						},
					},
				},
			},
		},
		{
			desc: "with list",
			input: `
			<!h> 
				{there}
			<!/>

			<h there="meme"/>
			`,
			output: []Element{
				{
					Name: "text",
					Attributes: Attribs{
						"text": {"meme"},
					},
					Children: []Element{},
					prefabData: []prefabData{
						{
							Name:   "there",
							Target: "text",
							Idx:    -2,
						},
					},
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			p.ClearPrefabs()
			div, err := p.Parse([]byte(tC.input))
			if !tC.err.SameSurface(err) {
				t.Error(p.Err)
				t.Error(string(p.Ch))
				return
			}

			if p.Failed() {
				return
			}

			core.TestEqual(t, div.Children, tC.output)

		})
	}
}

func TestPrefabDef(t *testing.T) {
	p := NParser(nil)
	p.AddDefinitions("div")
	err := p.AddPrefabs([]byte(`<!ff><!/>`))
	if err != nil {
		t.Error(err)
		return
	}
	p.RemovePrefabs("ff")
	testCases := []struct {
		desc   string
		input  string
		output pr
		err    sterr.Err
	}{
		{
			desc:  "empthy",
			input: `<!prefab><!/>`,
			output: pr{
				"prefab": Element{
					Name:       "prefab",
					Attributes: Attribs{},
				},
			},
		},
		{
			desc: "Ident",
			input: `<!prefab>
				<div h={}/>
			<!/>`,
			err: ErrPrefab.Ident,
		},
		{
			desc: "incomplete prefabe",
			input: `<!prefab>
				<div h={`,
			err: ErrAttrib.Incomplete,
		},
		{
			desc: "incomplete list prefab",
			input: `<!prefab>
				<div h=[{`,
			err: ErrAttrib.Incomplete,
		},
		{
			desc:  "outside",
			input: `<div h={h}/>`,
			err:   ErrPrefab.Outside,
		},
		{
			desc:  "outside text",
			input: `{h}`,
			err:   ErrPrefab.Outside,
		},
		{
			desc:  "duplicate",
			input: `<!prefab><!/><!prefab><!/>`,
			err:   ErrPrefab.Shadow,
		},
		{
			desc:  "attributes",
			input: `<!prefab h="h">`,
			err:   ErrPrefab.Attributes,
		},
		{
			desc:  "incomplete",
			input: `<!`,
			err:   ErrDiv.Incomplete,
		},
		{
			desc:  "extra closure",
			input: `<!/>`,
			err:   ErrDiv.ExtraClosure,
		},
		{
			desc: "template",
			input: `
<!prefab>
	<div hello={mel} ffl=["gl" {ghl}]/>
<!/>
			`,
			output: pr{
				"prefab": Element{
					Name:       "prefab",
					Attributes: Attribs{},
					Children: []Element{
						{
							Name: "div",
							Attributes: Attribs{
								"ffl": {"gl", ""},
							},
							prefabData: []prefabData{
								{
									Name:   "mel",
									Target: "hello",
									Idx:    -1,
								},
								{
									Name:   "ghl",
									Target: "ffl",
									Idx:    1,
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			p.ClearPrefabs()
			_, err := p.Parse([]byte(tC.input))
			if !tC.err.SameSurface(err) {
				t.Error(p.Err)
				t.Error(sterr.ReadTrace(p.Err))
				t.Error(string(p.Ch))
				return
			}

			if p.Failed() {
				return
			}

			if !reflect.DeepEqual(p.prefabs, tC.output) {
				t.Error(p.prefabs)
			}
		})
	}
}

func TestParse(t *testing.T) {
	p := NParser(nil)
	p.AddDefinitions("div", "fiv", "giv")
	testCases := []struct {
		desc   string
		input  string
		output []Element
		err    sterr.Err
	}{
		{
			desc: "simple",
			input: `
		<#>comment<#>
		<div>
			<fiv>
				<giv/>
				hello
				<giv/>
			</>
		</>
					`,
			output: []Element{
				{
					Name:       "div",
					Attributes: Attribs{},
					Children: []Element{
						{
							Name:       "fiv",
							Attributes: Attribs{},
							Children: []Element{
								{
									Name:       "giv",
									Attributes: Attribs{},
								},
								{
									Name:       "text",
									Attributes: Attribs{"text": {"hello"}},
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
			desc:  "just text",
			input: `hello `,
			output: []Element{
				{
					Name: "text",
					Attributes: Attribs{
						"text": {"hello"},
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
			desc:  "comment after hash",
			input: `<#`,
			err:   ErrComment.AfterHash,
		},
		{
			desc:  "comment not closed",
			input: `<#>asd`,
			err:   ErrComment.NotClosed,
		},
		{
			desc:  "after slash",
			input: `<div></`,
			err:   ErrDiv.AfterSlash,
		},
		{
			desc:  "missing closure",
			input: `<div>`,
			err:   ErrDiv.MissingClosure,
		},
		{
			desc:  "extra closure",
			input: `<div></></>`,
			err:   ErrDiv.ExtraClosure,
		},
		{
			desc:  "div error",
			input: `<div/ >`,
			err:   ErrDiv.AfterSlash,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			p.ClearPrefabs()
			div, err := p.Parse([]byte(tC.input))
			if !tC.err.SameSurface(err) {
				t.Error(p.Err)
				t.Error(string(p.Ch), p.stack)
				return
			}

			if p.Failed() {
				return
			}

			if !reflect.DeepEqual(div.Children, tC.output) {
				t.Error(div.Children, p.stack)
			}
		})
	}
}

func TestDiv(t *testing.T) {
	p := NParser(nil)
	p.AddDefinitions("niv")
	p.ClearDefinitions()
	p.AddDefinitions("div", "fiv", "giv", "riv")
	p.RemoveDefinitions("riv")

	testCases := []struct {
		desc   string
		input  string
		output []Element
		err    sterr.Err
	}{
		{
			desc:  "simple",
			input: `<div hello="hello" krr=["asd" "asd"]/>`,
			output: []Element{
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
			output: []Element{
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
			err:   ErrUnknown,
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
		{
			desc:  "attrib err",
			input: `<div h,"f"/>`,
			err:   ErrAttrib.Assignmant,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			p.Restart([]byte(tC.input))
			p.Advance()
			p.Advance()
			p.element(false)
			if !tC.err.SameSurface(p.Err) {
				t.Error(p.Err)
				return
			}

			if p.Failed() {
				return
			}

			if !reflect.DeepEqual(p.root.Children, tC.output) && !reflect.DeepEqual([]Element(p.stack), tC.output) {
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
			input: `hello=["hello"] `,
			output: Attribs{
				"hello": {"hello"},
			},
		},
		{
			desc:  "list",
			input: `hello=["hello\xkk"] `,
			err:   ErrEscape.Illegal,
		},
		{
			desc:  "list",
			input: `hello=[{hello}] `,
			err:   ErrPrefab.Outside,
		},
		{
			desc:  "incomplete",
			input: ``,
			err:   ErrDiv.Incomplete,
		},
		{
			desc:  "long list",
			input: `hello=["hello" "fl" "gg" "mm"] `,
			output: Attribs{
				"hello": {"hello", "fl", "gg", "mm"},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			p.Restart([]byte(tC.input))
			p.parsed = NDiv()
			p.attribute()
			if !tC.err.SameSurface(p.Err) {
				t.Error(p.Err)
				t.Error(string(p.Ch))
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
		ending        byte
		omit          bool
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
			desc:  "invalid escape Ident",
			input: "\\kFF\"",
			err:   ErrEscape.InvalidIdent,
		},
		{
			desc:   "navigation runes",
			input:  "\t\r\n\"",
			output: "   ",
		},
		{
			desc:   "template string",
			input:  "{hello} {{hello}\"",
			output: "{hello} {hello}",
		},
		{
			desc:  "template string",
			input: "{",
			err:   ErrStringNotTerminated,
		},
		{
			desc:   "concat space",
			omit:   true,
			input:  "a \\  b \t\"",
			output: "a  b",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			p.Restart([]byte(tC.input))
			p.inPrefab = true
			if tC.ending == 0 {
				tC.ending = '"'
			}
			p.string(tC.ending, tC.omit)
			if !tC.err.SameSurface(p.Err) {
				t.Error(p.Err)
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
