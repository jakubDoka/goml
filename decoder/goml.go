package goml

import (
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
	ErrReport = sterr.New("located at %d:%d")
	ErrByte   = sterr.New("unexpected byte outside div and string definition, if you want to add text to element put it into \" \"")
)

// ErrDiv stores errors related to div
var ErrDiv = struct {
	Incomplete, Identifier, Unknown, AfterIdent, AfterSlash, ExtraClosure sterr.Err
}{
	sterr.New("incomplete div definition"),
	sterr.New("'<' must be folloved by div identifier when defining div"),
	sterr.New("use of unknown div"),
	sterr.New("ident of div has to be folloved by ' ' or '/' or '>'"),
	sterr.New("'/' must alway be folloved by '>' if its part of div definition"),
	sterr.New("found closing syntax but there is no div to close"),
}

// ErrAttrib stores attribute related errors
var ErrAttrib = struct {
	Assignmant, Incomplete, ValueStart, ExtraSpace, BetweenByte, ListIncomplete sterr.Err
}{
	sterr.New("attribute can be assigned with '=' or set to true by following it with ' '"),
	sterr.New("attribute definition is incomplete"),
	sterr.New("'=' can be followed only by '['(list definition) or '\"'(single value)"),
	sterr.New("extra space after a last list value is not allowed"),
	sterr.New("unexpected byte in-between byte in list definition, use just one ' ' to separate values"),
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

// Parse ...
func (p *Parser) Parse(source []byte) (Div, error) {
	p.Restart(source)
	for !p.Failed() && p.SkipSpace() {
		switch p.ch {
		case '<':
			if !p.Advance() {
				p.Error(ErrDiv.Incomplete)
				break
			}
			switch p.ch {
			case '/':
				p.DivEnd()
			case '!':
				//p.ParsePrefab()
			default:
				p.i--
				p.Div()

			}
		case '"':
			//p.TextDiv()
		default:
			p.Error(ErrByte)
		}
	}

	return p.root, p.err
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

// Restart restarts parser state for another parsing
func (p *Parser) Restart(source []byte) {
	p.source = source
	p.i = -1
	p.line = 0
	p.lineStart = 0
	p.err = nil
	p.root = NDiv()
	p.stack = p.stack[:0]
}

// DivEnd closes a p.Current() div, also verifies that closing syntax is correct
func (p *Parser) DivEnd() {
	if !p.Check('>') {
		p.Error(ErrDiv.AfterSlash)
		return
	}
	if p.stack.CanPop() {
		p.Add(p.stack.Pop())
	} else {
		p.Error(ErrDiv.ExtraClosure)
	}
}

// Div parses a div definition with its attributes, if div contains children, it will push it to stack
// othervise bits is pushed to p.Current()
func (p *Parser) Div() {
	p.parsed = NDiv()
	p.parsed.Name = string(p.Ident())

	if p.parsed.Name == "" {
		p.Error(ErrDiv.Identifier)
		return
	}

	if !p.defined[p.parsed.Name] {
		p.Error(ErrDiv.Unknown)
		return
	}

	for !p.Failed() {
		switch p.ch {
		case ' ':
			p.Attribute()
			continue
		case '/':
			if !p.Check('>') {
				p.Error(ErrDiv.AfterSlash)
				return
			}
			p.Add(p.parsed)
		case '>':
			p.stack = append(p.stack, p.parsed)
		default:
			p.Error(ErrDiv.AfterIdent)
		}
		return
	}

}

// Attribute parses one attribute of div
func (p *Parser) Attribute() {
	p.attribIdent = string(p.Ident())
	switch p.ch {
	case '=':
		p.Value()
	case ' ':
		p.parsed.Attributes[p.attribIdent] = append(p.parsed.Attributes[p.attribIdent], "true")
	default:
		p.Error(ErrAttrib.Assignmant)
	}
}

// Value parses attribute value, whether it is list:
//
//	["a", "b", "c"]
//
// or just simple string, it will append to current p.attribIdent of p.parsed.Attributes
func (p *Parser) Value() {
	if !p.Advance() {
		p.Error(ErrAttrib.Incomplete)
		return
	}

	switch p.ch {
	case '"':
		p.String()
		if p.Failed() {
			return
		}
		p.parsed.Attributes[p.attribIdent] = append(p.parsed.Attributes[p.attribIdent], string(p.stringBuff))
		return
	case '[':
		p.List()
		return
	default:
		p.Error(ErrAttrib.ValueStart)
		return
	}
}

// List parses list literal
func (p *Parser) List() {
	list := p.parsed.Attributes[p.attribIdent]
	for !p.Failed() {
		switch p.ch {
		case ' ', '[':
			if !p.Advance() {
				p.Error(ErrAttrib.ListIncomplete)
				return
			}

			switch p.ch {
			case ' ':
				p.Error(ErrAttrib.ExtraSpace)
				return
			case '"':
			default:
				p.Error(ErrAttrib.BetweenByte)
			}

			p.String()
			if p.Failed() {
				return
			}
			list = append(list, string(p.stringBuff))
		case ']':
			p.parsed.Attributes[p.attribIdent] = list
			p.Advance()
			return
		default:
			p.Error(ErrAttrib.BetweenByte)
			return
		}
	}
}

/*
func (p *Parser) Clear() {
	for k := range p.defined {
		delete(p.defined, k)
	}
	for k := range p.prefabs {
		delete(p.prefabs, k)
	}
}*/

// Failed returns whether error happened
func (p *Parser) Failed() bool {
	return p.err != nil
}

// Error sets p.err and adds the line info
func (p *Parser) Error(err sterr.Err) {
	p.err = err.Wrap(ErrReport.Args(p.line, p.i-p.lineStart))
}

// Check returns true if p.Advance succeeds and p.ch == b
func (p *Parser) Check(b byte) bool {
	return p.Advance() && p.ch == b
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
}

// NDiv creates ready-to-use div
func NDiv() Div {
	return Div{
		Attributes: Attribs{},
	}
}
