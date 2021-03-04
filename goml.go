package goml

import (
	"strings"

	"github.com/jakubDoka/goml/core"
	"github.com/jakubDoka/goml/goss"
	"github.com/jakubDoka/sterr"
)

/*imp(
	github.com/jakubDoka/gogen/templates
)*/

/*gen(
	templates.Stack<Element, DivStack>
)*/

// error variants
var (
	ErrUnknown = sterr.New("use of unknown identifier")
	ErrStyle   = sterr.New("error when parsing style")
)

// ErrComment store comment related errors
var ErrComment = struct {
	AfterHash, NotClosed sterr.Err
}{
	sterr.New("'#' has to be folloved by '>' when declaring comment"),
	sterr.New("comment is not terminated"),
}

// ErrDiv stores errors related to div
var ErrDiv = struct {
	Incomplete, Identifier, AfterIdent, AfterSlash, ExtraClosure, MissingClosure sterr.Err
}{
	sterr.New("incomplete element definition"),
	sterr.New("'<' must be folloved by element identifier when defining div"),
	sterr.New("Ident of element has to be folloved by ' ' or '/' or '>'"),
	sterr.New("'/' must alway be folloved by '>' if its part of element definition"),
	sterr.New("found closing syntax but there is no element to close"),
	sterr.New("some elements are not closed"),
}

// ErrPrefab stores prefab related errors
var ErrPrefab = struct {
	Shadow, Outside, Ident, Attributes sterr.Err
}{
	sterr.New("prefab cannot shadow existing element or prefab"),
	sterr.New("prefab syntax outside a prefab block is not allowed"),
	sterr.New("only identifier is allowed between '{}', found '%s' witch cannot be part of ident"),
	sterr.New("prefab definition cannot have attributes"),
}

// ErrAttrib stores attribute related errors
var ErrAttrib = struct {
	Assignmant, Incomplete, ValueStart, ExtraSpace, BetweenByte, ListIncomplete sterr.Err
}{
	sterr.New("attribute can be assigned with '=' or set to true by following it with ' '"),
	sterr.New("attribute definition is incomplete"),
	sterr.New("'=' can be followed only by '['(list definition) or '\"'(single value)"),
	sterr.New("extra space after a last list value is not allowed"),
	sterr.New("unexpected in-between byte in list definition, use just one ' ' to separate values"),
	sterr.New("list is incomplete"),
}

// CommentEnd is group of comment closing bytes
var CommentEnd = []byte("<#>")

// Parser takes a goml syntax and parses it into DivTree
type Parser struct {
	gs      *goss.Parser
	stack   DivStack
	defined map[string]bool
	prefabs map[string]Element

	attribIdent  string
	root, parsed Element
	stringBuff   []rune
	styleBuff    []byte
	inPrefab     bool

	parser
}

// NParser creates ready-to-use Parser
func NParser(sp *goss.Parser) *Parser {
	return &Parser{
		defined: map[string]bool{},
		prefabs: map[string]Element{},
		gs:      sp,
	}
}

// AddDefinitions add definitions into parser, all names will be considered
// as valid element identifiers
func (p *Parser) AddDefinitions(names ...string) {
	for _, name := range names {
		p.defined[name] = true
	}
}

// RemoveDefinitions removes definitions from defSet
func (p *Parser) RemoveDefinitions(names ...string) {
	for _, name := range names {
		delete(p.defined, name)
	}
}

// ClearDefinitions clears all definitions so that no element is valid
func (p *Parser) ClearDefinitions() {
	for name := range p.defined {
		delete(p.defined, name)
	}
}

// ClearPrefabs clears all prefabs so they can be redefined
func (p *Parser) ClearPrefabs() {
	for name := range p.prefabs {
		delete(p.prefabs, name)
	}
}

// AddPrefabs adds prefabs from Source
func (p *Parser) AddPrefabs(Source []byte) error {
	_, err := p.Parse(Source)
	return err
}

// RemovePrefabs removes all prefabs under given identifiers
func (p *Parser) RemovePrefabs(names ...string) {
	for _, n := range names {
		delete(p.prefabs, n)
	}
}

// Restart restarts parser state for another parsing
func (p *Parser) Restart(Source []byte) {
	p.parser.Restart(Source)
	p.root = NDiv()
	p.stack = p.stack[:0]
	p.inPrefab = false
}

// Parse ...
func (p *Parser) Parse(Source []byte) (Element, error) {
	p.Restart(Source)
	for p.SkipSpace() && !p.Failed() {
		switch p.Ch {
		case '<':
			if p.AdvanceOr(ErrDiv.Incomplete) {
				break
			}

			switch p.Ch {
			case '/':
				if !p.elementEnd(false) {
					break
				}
			case '!':
				if p.AdvanceOr(ErrDiv.Incomplete) {
					break
				}

				if p.Ch == '/' {
					if !p.elementEnd(true) {
						break
					}
					continue
				}

				p.inPrefab = true
				if !p.element(true) {
					break
				}
			case '#':
				if p.Check('>', ErrComment.AfterHash) {
					break
				}
				for {
					equal, ok := p.CheckSlice(CommentEnd)
					if !ok {
						p.Error(ErrComment.NotClosed)
						break
					}
					if equal {
						p.Advance()
						p.Advance()
						p.Advance()
						break
					} else {
						p.Advance()
					}
				}
			default:
				if !p.element(false) {
					break
				}
			}
		default:
			p.textElement()
		}
	}

	if len(p.stack) != 0 && p.Err == nil {
		p.Error(ErrDiv.MissingClosure)
	}

	return p.root, p.Err
}

// textElement parses a text paragraph into element with text attribute
func (p *Parser) textElement() bool {
	p.parsed = NDiv()
	p.parsed.Name = "text"
	p.attribIdent = "text"
	p.Degrade()
	if !p.string('<', true) {
		return false
	}
	p.parsed.Attributes[p.attribIdent] = []string{string(p.stringBuff)}

	if p.Peek() {
		p.Degrade()
		p.Degrade()
	}

	p.add(p.parsed)
	return true
}

// elementEnd closes a p.current() div, also verifies that closing syntax is correct
func (p *Parser) elementEnd(prefab bool) bool {
	if p.Check('>', ErrDiv.AfterSlash) {
		return false
	}

	if p.stack.CanPop() {
		d := p.stack.Pop()
		if p.inPrefab && prefab {
			p.prefabs[d.Name] = d
			p.inPrefab = false
		} else {
			p.add(d)
		}
		return true
	}

	p.Error(ErrDiv.ExtraClosure)
	return false
}

// Element parses a element definition with its attributes, if element contains children, it will push it to stack
// othervise bits is pushed to p.current()
func (p *Parser) element(isPrefab bool) bool {
	p.parsed = NDiv()
	p.parsed.Name = string(p.Ident())

	if p.parsed.Name == "" {
		p.Error(ErrDiv.Identifier)
		return false
	}

	prefab, pok := p.prefabs[p.parsed.Name]
	dok := p.defined[p.parsed.Name]
	if p.inPrefab {
		if pok {
			p.Error(ErrPrefab.Shadow)
			return false
		}
	} else {
		if !pok && !dok {
			p.Error(ErrUnknown)
			return false
		}
	}

	for {
		switch p.Ch {
		case ' ':
			if isPrefab {
				p.Error(ErrPrefab.Attributes)
				return false
			}

			if !p.attribute() {
				return false
			}
			continue
		case '/':
			if p.Check('>', ErrDiv.AfterSlash) {
				return false
			}

			if pok {
				prefab = p.prefabs[p.parsed.Name].create(p.parsed.Attributes)
				for _, ch := range prefab.Children {
					p.add(ch)
				}
			} else {
				p.add(p.parsed)
			}
		case '>':
			p.stack.Push(p.parsed)
		default:
			p.Error(ErrDiv.AfterIdent)
			return false
		}
		return true
	}
}

// attribute parses one attribute of div
func (p *Parser) attribute() bool {
	if p.AdvanceOr(ErrDiv.Incomplete) {
		return false
	}
	p.attribIdent = string(p.Ident())
	switch p.Ch {
	case '=':
		return p.value()
	case ' ':
		p.parsed.Attributes[p.attribIdent] = append(p.parsed.Attributes[p.attribIdent], "true")
		return true
	default:
		p.Error(ErrAttrib.Assignmant)
		return false
	}
}

// value parses attribute value, whether it is list:
//
//	["a", "b", "c"]
//
// or just simple string, it will append to current p.attribIdent of p.parsed.Attributes
func (p *Parser) value() bool {
	if p.AdvanceOr(ErrAttrib.Incomplete) {
		return false
	}

	switch p.Ch {
	case '"':
		if p.string('"', false) {
			p.parsed.Attributes[p.attribIdent] = []string{string(p.stringBuff)}

			if p.gs != nil && p.attribIdent == "style" {
				p.styleBuff = p.styleBuff[:0]
				for _, r := range p.stringBuff {
					p.styleBuff = append(p.styleBuff, byte(r))
				}
				style, err := p.gs.Style(p.styleBuff)
				if err != nil {
					p.Err = ErrStyle.Wrap(p.ReportError().Wrap(err))
					return false
				}
				p.parsed.Style = style
			}
			return true
		}
		return false
	case '{':
		if p.AdvanceOr(ErrAttrib.Incomplete) {
			return false
		}
		return p.template(&p.parsed, wholeTemplate)
	case '[':
		return p.list()
	default:
		p.Error(ErrAttrib.ValueStart)
		return false
	}
}

// list parses list literal
func (p *Parser) list() bool {
	list := p.parsed.Attributes[p.attribIdent]
	for {
		switch p.Ch {
		case ' ', '[':
			if p.AdvanceOr(ErrAttrib.ListIncomplete) {
				return false
			}

			switch p.Ch {
			case ' ':
				p.Error(ErrAttrib.ExtraSpace)
				return false
			case '"':
				if !p.string('"', false) {
					return false
				}
				list = append(list, string(p.stringBuff))
			case '{':
				if p.AdvanceOr(ErrAttrib.Incomplete) {
					return false
				}
				if !p.template(&p.parsed, len(list)) {
					return false
				}
				list = append(list, "") // place a placeholder
			default:
				p.Error(ErrAttrib.BetweenByte)
				return false
			}
		case ']':
			p.parsed.Attributes[p.attribIdent] = list

			return !p.AdvanceOr(ErrAttrib.Incomplete)
		default:
			p.Error(ErrAttrib.BetweenByte)
			return false
		}
	}
}

// template parses a prefab template parameter and saves it to prefab div
func (p *Parser) template(target *Element, idx int) bool {
	if !p.inPrefab {
		p.Error(ErrPrefab.Outside)
		return false
	}

	name := string(p.Ident())
	if name == "" || p.Ch != '}' {
		p.Error(ErrPrefab.Ident.Args(string(p.Ch)).Trace(4))
		//panic(p.Err)
		return false
	}

	target.prefabData = append(target.prefabData, prefabData{
		Target: p.attribIdent,
		Name:   name,
		Idx:    idx,
	})

	return !p.AdvanceOr(ErrDiv.Incomplete)
}

// add adds child to current div
func (p *Parser) add(d Element) {
	c := p.current()
	c.Children = append(c.Children, d)
}

// current returns Element on stack top
func (p *Parser) current() *Element {
	if p.stack.CanPop() {
		return p.stack.Top()
	}
	return &p.root
}

// Attribs ...
type Attribs = map[string][]string

// Element is representation goml element
type Element struct {
	Name       string
	Attributes Attribs
	Style      goss.Style
	Children   []Element
	prefabData []prefabData
}

// NDiv creates ready-to-use div
func NDiv() Element {
	return Element{
		Attributes: Attribs{},
	}
}

// Create creates template
func (d Element) create(atr Attribs) Element {
	// copy and create children
	nch := make([]Element, len(d.Children))
	for i := range nch {
		nch[i] = d.Children[i].create(atr)
	}
	d.Children = nch

	// copy attributes
	nat := make(Attribs, len(d.Attributes))
	for k, v := range d.Attributes {
		nat[k] = v
	}
	d.Attributes = nat

	// fill prefab data
	for _, pd := range d.prefabData {
		val, ok := atr[pd.Name]
		if !ok {
			continue
		}

		// we are ignoring other values if supplied unless its a whole value
		switch pd.Idx {
		case wholeTemplate:
			d.Attributes[pd.Target] = val
		case stringTemplate:
			d.Attributes[pd.Target][0] = strings.Replace(d.Attributes[pd.Target][0], "{"+pd.Name+"}", val[0], 1)
		default:
			d.Attributes[pd.Target][pd.Idx] = val[0]
		}
	}

	return d
}

// prefabData related constants
const (
	wholeTemplate  = -1
	stringTemplate = -2
)

// prefabData stores data for prefab generation
type prefabData struct {
	Name, Target string
	Idx          int
}

type parser struct {
	core.Parser
}
