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
	templates.Stack<Div, DivStack>
)*/

// Error variants
var (
	ErrReport  = sterr.New("located at %d:%d")
	ErrByte    = sterr.New("unexpected byte outside div and string definition, if you want to add text to element put it into \" \"")
	ErrUnknown = sterr.New("use of unknown identifier")
)

// ErrDiv stores errors related to div
var ErrDiv = struct {
	Incomplete, Identifier, AfterIdent, AfterSlash, ExtraClosure sterr.Err
}{
	sterr.New("incomplete div definition"),
	sterr.New("'<' must be folloved by div identifier when defining div"),
	sterr.New("ident of div has to be folloved by ' ' or '/' or '>'"),
	sterr.New("'/' must alway be folloved by '>' if its part of div definition"),
	sterr.New("found closing syntax but there is no div to close"),
}

// ErrPrefab stores prefab related errors
var ErrPrefab = struct {
	Shadow, Outside, Ident, Attributes sterr.Err
}{
	sterr.New("prefab cannot shadow existing div or prefab"),
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
	prefabs map[string]Div

	attribIdent        string
	root, parsed       Div
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
		prefabs: map[string]Div{},
	}
}

// AddDefinitions add definitions into parser, all names will be considered
// as valid div element identifiers
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

// ClearDefinitions clears all definitions so that no div is valid
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
func (p *Parser) Parse(source []byte) (Div, error) {
	p.Restart(source)
	for p.SkipSpace() && !p.Failed() {
		switch p.ch {
		case '<':
			if p.AdvanceOr(ErrDiv.Incomplete) {
				break
			}

			switch p.ch {
			case '/':
				if !p.DivEnd() {
					break
				}
			case '!':
				if p.AdvanceOr(ErrDiv.Incomplete) {
					break
				}

				if p.ch == '/' {
					if !p.DivEnd() {
						break
					}
					continue
				}

				p.inPrefab = true
				if !p.Div(true) {
					break
				}
			default:
				if !p.Div(false) {
					break
				}
			}
		default:
			p.TextDiv()
		}
	}

	return p.root, p.err
}

// TextDiv parses a text paragraph into div with text attribute
func (p *Parser) TextDiv() bool {
	p.parsed = NDiv()
	p.parsed.Name = "text"
	p.attribIdent = "text"
	p.Degrade()
	if !p.String('<', true) {
		return false
	}
	p.parsed.Attributes[p.attribIdent] = []string{string(p.stringBuff)}

	p.Add(p.parsed)
	p.Degrade()
	p.Degrade()
	return true
}

// DivEnd closes a p.Current() div, also verifies that closing syntax is correct
func (p *Parser) DivEnd() bool {
	if p.Check('>', ErrDiv.AfterSlash) {
		return false
	}

	if p.stack.CanPop() {
		d := p.stack.Pop()
		if p.inPrefab {
			p.prefabs[d.Name] = d
			p.inPrefab = false
		} else {
			p.Add(d)
		}
		return true
	}

	p.Error(ErrDiv.ExtraClosure)
	return false
}

// Div parses a div definition with its attributes, if div contains children, it will push it to stack
// othervise bits is pushed to p.Current()
func (p *Parser) Div(isPrefab bool) bool {
	p.Degrade()
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
		switch p.ch {
		case ' ':
			if isPrefab {
				p.Error(ErrPrefab.Attributes)
				return false
			}

			if !p.Attribute() {
				return false
			}
			continue
		case '/':
			if p.Check('>', ErrDiv.AfterSlash) {
				return false
			}

			if pok {
				prefab = p.prefabs[p.parsed.Name].Create(p.parsed.Attributes)
				for _, ch := range prefab.Children {
					p.Add(ch)
				}
			} else {
				p.Add(p.parsed)
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

// Attribute parses one attribute of div
func (p *Parser) Attribute() bool {
	p.attribIdent = string(p.Ident())
	switch p.ch {
	case '=':
		return p.Value()
	case ' ':
		p.parsed.Attributes[p.attribIdent] = append(p.parsed.Attributes[p.attribIdent], "true")
		return true
	default:
		p.Error(ErrAttrib.Assignmant)
		return false
	}
}

// Value parses attribute value, whether it is list:
//
//	["a", "b", "c"]
//
// or just simple string, it will append to current p.attribIdent of p.parsed.Attributes
func (p *Parser) Value() bool {
	if p.AdvanceOr(ErrAttrib.Incomplete) {
		return false
	}

	switch p.ch {
	case '"':
		if p.String('"', false) {
			p.parsed.Attributes[p.attribIdent] = append(p.parsed.Attributes[p.attribIdent], string(p.stringBuff))
			return true
		}
		return false
	case '{':
		return p.Template(&p.parsed, WholeTemplate)
	case '[':
		return p.List()
	default:
		p.Error(ErrAttrib.ValueStart)
		return false
	}
}

// List parses list literal
func (p *Parser) List() bool {
	list := p.parsed.Attributes[p.attribIdent]
	for {
		switch p.ch {
		case ' ', '[':
			if p.AdvanceOr(ErrAttrib.ListIncomplete) {
				return false
			}

			switch p.ch {
			case ' ':
				p.Error(ErrAttrib.ExtraSpace)
				return false
			case '"':
				if !p.String('"', false) {
					return false
				}
				list = append(list, string(p.stringBuff))
			case '{':
				if !p.Template(&p.parsed, len(list)) {
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

// Template parses a prefab template parameter and saves it to prefab div
func (p *Parser) Template(target *Div, idx int) bool {
	if !p.inPrefab {
		p.Error(ErrPrefab.Outside)
		return false
	}

	name := string(p.Ident())
	if name == "" || p.ch != '}' {
		p.Error(ErrPrefab.Ident)
		return false
	}

	target.PrefabData = append(target.PrefabData, PrefabData{
		Target: p.attribIdent,
		Name:   name,
		Idx:    idx,
	})

	return !p.AdvanceOr(ErrDiv.Incomplete)
}

// Failed returns whether error happened
func (p *Parser) Failed() bool {
	return p.err != nil
}

// Error sets p.err and adds the line info
func (p *Parser) Error(err sterr.Err) {
	p.err = err.Wrap(ErrReport.Args(p.line, p.i-p.lineStart))
}

// Check returns false if p.Advance succeeds and p.ch == b
// othervise it rises error
func (p *Parser) Check(b byte, err sterr.Err) bool {
	ok := p.Advance() && p.ch == b
	if !ok {
		p.Error(err)
	}
	return !ok
}

// SkipSpace ignores all invisible characters until it finds visible one
// if there is new line character, it updates the p.line and p.lineStart
//
// returns false if Advance fails
func (p *Parser) SkipSpace() bool {
	for p.Advance() {
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

// Ident reads ident and returns slice where it is located
func (p *Parser) Ident() []byte {
	start := p.i + 1
	for p.Advance() && str.IsIdent(p.ch) || p.ch == '.' {
	}
	return p.source[start:p.i]
}

// AdvanceOr raises error if advancement fails, return value of p.Advance is inverted
func (p *Parser) AdvanceOr(err sterr.Err) bool {
	ok := p.Advance()
	if !ok {
		p.Error(err)
	}
	return !ok
}

// Advance calls p.Peek acd increases i
func (p *Parser) Advance() bool {
	ok := p.Peek()
	if ok {
		p.i++
	}
	return ok
}

// Peek stores next byte in p.ch, returns true is action wos successfull
func (p *Parser) Peek() bool {
	if p.i+1 >= len(p.source) {
		return false
	}
	p.ch = p.source[p.i+1]
	return true
}

// Degrade goes one byte back, opposite of p.Advance
func (p *Parser) Degrade() {
	p.i--
	p.ch = p.source[p.i]
}

// Add adds child to current div
func (p *Parser) Add(d Div) {
	c := p.Current()
	c.Children = append(c.Children, d)
}

// Current returns Div on stack top
func (p *Parser) Current() *Div {
	if p.stack.CanPop() {
		return p.stack.Top()
	}
	return &p.root
}

// Attribs ...
type Attribs = map[string][]string

// Div is representation goml element
type Div struct {
	Name       string
	Attributes Attribs
	Children   []Div
	PrefabData []PrefabData
}

// NDiv creates ready-to-use div
func NDiv() Div {
	return Div{
		Attributes: Attribs{},
	}
}

// Create creates template
func (d Div) Create(atr Attribs) Div {
	// copy and create children
	nch := make([]Div, len(d.Children))
	for i := range nch {
		nch[i] = d.Children[i].Create(atr)
	}
	d.Children = nch

	// copy attributes
	nat := make(Attribs, len(d.Attributes))
	for k, v := range d.Attributes {
		nat[k] = v
	}
	d.Attributes = nat

	// fill prefab data
	for _, pd := range d.PrefabData {
		val, ok := atr[pd.Name]
		if !ok {
			continue
		}

		// we are ignoring other values if supplied unless its a whole value
		switch pd.Idx {
		case WholeTemplate:
			d.Attributes[pd.Target] = val
		case StringTemplate:
			d.Attributes[pd.Target][0] = strings.Replace(d.Attributes[pd.Target][0], "{"+pd.Name+"}", val[0], 1)
		default:
			d.Attributes[pd.Target][pd.Idx] = val[0]
		}
	}

	return d
}

// PrefabData related constants
const (
	WholeTemplate  = -1
	StringTemplate = -2
)

// PrefabData stores data for prefab generation
type PrefabData struct {
	Name, Target string
	Idx          int
}
