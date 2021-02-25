package goml

import (
	"strings"

	"github.com/jakubDoka/gogen/str"
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
	ErrReport  = sterr.New("located at %d:%d")
	ErrUnknown = sterr.New("use of unknown identifier")
)

// ErrDiv stores errors related to div
var ErrDiv = struct {
	Incomplete, Identifier, AfterIdent, AfterSlash, ExtraClosure sterr.Err
}{
	sterr.New("incompleteelementdefinition"),
	sterr.New("'<' must be folloved by element identifier when defining div"),
	sterr.New("ident of element has to be folloved by ' ' or '/' or '>'"),
	sterr.New("'/' must alway be folloved by '>' if its part of element definition"),
	sterr.New("found closing syntax but there is no element to close"),
}

// ErrPrefab stores prefab related errors
var ErrPrefab = struct {
	Shadow, Outside, ident, Attributes sterr.Err
}{
	sterr.New("prefab cannot shadow existing element or prefab"),
	sterr.New("prefab syntax outside a prefab block is not allowed"),
	sterr.New("only ident is allowed between '{}', spaces cannot be there"),
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

// Parser takes a goml syntax and parses it into DivTree
type Parser struct {
	stack   DivStack
	defined map[string]bool
	prefabs map[string]Element

	attribIdent        string
	root, parsed       Element
	source             []byte
	stringBuff         []rune
	i, line, lineStart int
	ch                 byte
	err                error
	inPrefab           bool
}

// NParser creates ready-to-use Parser
func NParser() *Parser {
	return &Parser{
		defined: map[string]bool{},
		prefabs: map[string]Element{},
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

// AddPrefabs adds prefabs from source
func (p *Parser) AddPrefabs(source []byte) error {
	_, err := p.Parse(source)
	return err
}

// RemovePrefabs removes all prefabs under given identifiers
func (p *Parser) RemovePrefabs(names ...string) {
	for _, n := range names {
		delete(p.prefabs, n)
	}
}

// Restart restarts parser state for another parsing
func (p *Parser) Restart(source []byte) {
	p.source = source
	p.i = -1
	p.line = 0
	p.lineStart = 0
	p.err = nil
	p.root = NDiv()
	p.stack = p.stack[:0]
	p.inPrefab = false
}

// Parse ...
func (p *Parser) Parse(source []byte) (Element, error) {
	p.Restart(source)
	for p.skipSpace() && !p.failed() {
		switch p.ch {
		case '<':
			if p.advanceOr(ErrDiv.Incomplete) {
				break
			}

			switch p.ch {
			case '/':
				if !p.elementEnd() {
					break
				}
			case '!':
				if p.advanceOr(ErrDiv.Incomplete) {
					break
				}

				if p.ch == '/' {
					if !p.elementEnd() {
						break
					}
					continue
				}

				p.inPrefab = true
				if !p.element(true) {
					break
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

	return p.root, p.err
}

// textElement parses a text paragraph into element with text attribute
func (p *Parser) textElement() bool {
	p.parsed = NDiv()
	p.parsed.Name = "text"
	p.attribIdent = "text"
	p.degrade()
	if !p.string('<', true) {
		return false
	}
	p.parsed.Attributes[p.attribIdent] = []string{string(p.stringBuff)}

	p.add(p.parsed)
	p.degrade()
	p.degrade()
	return true
}

// elementEnd closes a p.current() div, also verifies that closing syntax is correct
func (p *Parser) elementEnd() bool {
	if p.check('>', ErrDiv.AfterSlash) {
		return false
	}

	if p.stack.CanPop() {
		d := p.stack.Pop()
		if p.inPrefab {
			p.prefabs[d.Name] = d
			p.inPrefab = false
		} else {
			p.add(d)
		}
		return true
	}

	p.error(ErrDiv.ExtraClosure)
	return false
}

// Element parses a element definition with its attributes, if element contains children, it will push it to stack
// othervise bits is pushed to p.current()
func (p *Parser) element(isPrefab bool) bool {
	p.degrade()
	p.parsed = NDiv()
	p.parsed.Name = string(p.ident())

	if p.parsed.Name == "" {
		p.error(ErrDiv.Identifier)
		return false
	}

	prefab, pok := p.prefabs[p.parsed.Name]
	dok := p.defined[p.parsed.Name]
	if p.inPrefab {
		if pok {
			p.error(ErrPrefab.Shadow)
			return false
		}
	} else {
		if !pok && !dok {
			p.error(ErrUnknown)
			return false
		}
	}

	for {
		switch p.ch {
		case ' ':
			if isPrefab {
				p.error(ErrPrefab.Attributes)
				return false
			}

			if !p.attribute() {
				return false
			}
			continue
		case '/':
			if p.check('>', ErrDiv.AfterSlash) {
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
			p.error(ErrDiv.AfterIdent)
			return false
		}
		return true
	}
}

// attribute parses one attribute of div
func (p *Parser) attribute() bool {
	p.attribIdent = string(p.ident())
	switch p.ch {
	case '=':
		return p.value()
	case ' ':
		p.parsed.Attributes[p.attribIdent] = append(p.parsed.Attributes[p.attribIdent], "true")
		return true
	default:
		p.error(ErrAttrib.Assignmant)
		return false
	}
}

// value parses attribute value, whether it is list:
//
//	["a", "b", "c"]
//
// or just simple string, it will append to current p.attribIdent of p.parsed.Attributes
func (p *Parser) value() bool {
	if p.advanceOr(ErrAttrib.Incomplete) {
		return false
	}

	switch p.ch {
	case '"':
		if p.string('"', false) {
			p.parsed.Attributes[p.attribIdent] = append(p.parsed.Attributes[p.attribIdent], string(p.stringBuff))
			return true
		}
		return false
	case '{':
		return p.template(&p.parsed, wholeTemplate)
	case '[':
		return p.list()
	default:
		p.error(ErrAttrib.ValueStart)
		return false
	}
}

// list parses list literal
func (p *Parser) list() bool {
	list := p.parsed.Attributes[p.attribIdent]
	for {
		switch p.ch {
		case ' ', '[':
			if p.advanceOr(ErrAttrib.ListIncomplete) {
				return false
			}

			switch p.ch {
			case ' ':
				p.error(ErrAttrib.ExtraSpace)
				return false
			case '"':
				if !p.string('"', false) {
					return false
				}
				list = append(list, string(p.stringBuff))
			case '{':
				if !p.template(&p.parsed, len(list)) {
					return false
				}
				list = append(list, "") // place a placeholder
			default:
				p.error(ErrAttrib.BetweenByte)
				return false
			}
		case ']':
			p.parsed.Attributes[p.attribIdent] = list

			return !p.advanceOr(ErrAttrib.Incomplete)
		default:
			p.error(ErrAttrib.BetweenByte)
			return false
		}
	}
}

// template parses a prefab template parameter and saves it to prefab div
func (p *Parser) template(target *Element, idx int) bool {
	if !p.inPrefab {
		p.error(ErrPrefab.Outside)
		return false
	}

	name := string(p.ident())
	if name == "" || p.ch != '}' {
		p.error(ErrPrefab.ident)
		return false
	}

	target.prefabData = append(target.prefabData, prefabData{
		Target: p.attribIdent,
		Name:   name,
		Idx:    idx,
	})

	return !p.advanceOr(ErrDiv.Incomplete)
}

// failed returns whether error happened
func (p *Parser) failed() bool {
	return p.err != nil
}

// error sets p.err and adds the line info
func (p *Parser) error(err sterr.Err) {
	p.err = err.Wrap(ErrReport.Args(p.line, p.i-p.lineStart))
}

// check returns false if p.advance succeeds and p.ch == b
// othervise it rises error
func (p *Parser) check(b byte, err sterr.Err) bool {
	ok := p.advance() && p.ch == b
	if !ok {
		p.error(err)
	}
	return !ok
}

// skipSpace ignores all invisible characters until it finds visible one
// if there is new line character, it updates the p.line and p.lineStart
//
// returns false if advance fails
func (p *Parser) skipSpace() bool {
	for p.advance() {
		switch p.ch {
		case ' ', '\t':
		case '\n':
			p.line++
			p.lineStart = p.i + 1
		default:
			return true
		}
	}

	return false
}

// ident reads ident and returns slice where it is located
func (p *Parser) ident() []byte {
	start := p.i + 1
	for p.advance() && str.IsIdent(p.ch) || p.ch == '.' {
	}
	return p.source[start:p.i]
}

// advanceOr raises error if advancement fails, return value of p.advance is inverted
func (p *Parser) advanceOr(err sterr.Err) bool {
	ok := p.advance()
	if !ok {
		p.error(err)
	}
	return !ok
}

// advance calls p.peek acd increases i
func (p *Parser) advance() bool {
	ok := p.peek()
	if ok {
		p.i++
	}
	return ok
}

// peek stores next byte in p.ch, returns true is action wos successfull
func (p *Parser) peek() bool {
	if p.i+1 >= len(p.source) {
		return false
	}
	p.ch = p.source[p.i+1]
	return true
}

// degrade goes one byte back, opposite of p.advance
func (p *Parser) degrade() {
	p.i--
	p.ch = p.source[p.i]
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
