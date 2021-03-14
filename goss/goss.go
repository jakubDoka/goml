package goss

import (
	"strconv"
	"strings"

	"github.com/jakubDoka/goml/core"
	"github.com/jakubDoka/sterr"
)

/*imp(
	github.com/jakubDoka/gogen/templates
)*/

/*gen(
	templates.Stack<stl, Stack>
)*/

// Errors
var (
	ErrIdent           = sterr.New("expected identifier")
	ErrEmptyStyle      = sterr.New("empty style")
	ErrExpectedStart   = sterr.New("expected start of field(':') after ident")
	ErrNumber          = sterr.New("failed to parse number(%s)")
	ErrExpectedByte    = sterr.New("expected %s but found '%c'")
	ErrNoValues        = sterr.New("field '%s' in '%s' has no values")
	ErrFieldIncomplete = sterr.New("field is incomplete, it has to be terminated with ';'")
	ErrStyleIncomplete = sterr.New("style is incomplete, it has to be terminated with '}'")
	ErrExpectedValue   = sterr.New("expected value after ' '")
)

// Parser parses the goss "language"
type Parser struct {
	cField, cStyle string

	goml    bool
	val     interface{}
	valBuff []interface{}
	core.Parser
}

// Parse expects file full of styles that have declared names
func (p *Parser) Parse(source []byte) (Styles, error) {
	p.Restart(source)
	styles := Styles{}
	for p.SkipSpace() {
		ident := p.Ident()
		if ident == nil {
			p.Error(ErrIdent)
			break
		}
		val, ok := p.value().(Style)
		if !ok {
			break
		}
		styles[string(ident)] = val
	}
	return styles, p.Err
}

func (p *Parser) Style(source []byte) (Style, error) {
	p.Restart(source)
	p.goml = true
	p.Ch = '{'
	val, _ := p.value().(Style)
	return val, p.Err
}

func (p *Parser) value() interface{} {
	switch p.Ch {
	case '{':
		stl := Style{}
	o:
		for p.SkipSpace() {

			if p.Ch == '}' {
				return stl
			}
			ident := p.Ident()
			if ident == nil {
				p.Error(ErrIdent)
				return nil
			}
			id := string(ident)
			switch p.Ch {
			case '{':
				val := p.value()
				if val == nil {
					return nil
				}
				stl[id] = []interface{}{val}
			case ':':
				var val []interface{}
				for p.SkipSpace() {
					if p.Ch == ';' {
						stl[id] = val
						continue o
					}
					v := p.value()
					if v == nil {
						return nil
					}
					val = append(val, v)
					if _, ok := v.(Style); !ok {
						p.Degrade()
					}
				}
				p.Error(ErrFieldIncomplete)
				return nil
			default:
				p.Error(ErrExpectedByte.Args("':' or '{'", p.Ch))
				return nil
			}
		}
		if p.goml {
			return stl
		}
		p.Error(ErrStyleIncomplete)
		return nil
	default:
		if core.IsNumStart(p.Ch) {
			return p.number(p.Ch == '-')
		}
		ident := p.Ident()
		if ident == nil {
			p.Error(ErrExpectedValue)
			return nil
		}
		return string(ident)
	}
}

func (p *Parser) number(negative bool) (val interface{}) {
	slice := p.Number()
	if slice == nil {
		return nil
	}

	num := string(slice)
	if negative {
		num = "-" + num
	}

	var err error
	suffix := true
	switch p.Ch {
	case 'f':
		val, err = strconv.ParseFloat(num, 64)
	case 'i':
		val, err = strconv.Atoi(num)
	default:
		if strings.Contains(num, ".") {
			val, err = strconv.ParseFloat(num, 64)
		} else {
			val, err = strconv.Atoi(num)
		}
		suffix = false

		return
	}

	if err != nil {
		p.Error(ErrNumber.Args(err))
		return nil
	}

	if suffix {
		p.Advance()
	}

	return
}

type stl struct {
	name string
	stl  Style
}

// Style parses standalone ambiguous style
/*func (p *Parser) Style(source []byte) (Style, error) {
	p.Restart(source)
	p.cStyle = "inline"
	p.style(true)

	return p.parsed, p.Err
}

// Style
func (p *Parser) style(standalone bool) bool {
	p.parsed = Style{}
	for p.SkipSpace() {
		if p.Ch == ';' {
			return true
		}
		if !p.ident(&p.cField) {
			return false
		}

		if p.AdvanceOr(ErrIncomplete) {
			return false
		}

		p.valBuff = p.valBuff[:0]
	o:
		for {
			switch p.Ch {
			case ';':
				break o
			case ' ':
			default:
				p.Error(ErrExpectedByte.Args(' ', p.Ch))
				return false
			}

			if p.AdvanceOr(ErrIncomplete) {
				return false
			}
			if !p.value() {
				return false
			}

			p.valBuff = append(p.valBuff, p.val)
		}

		if len(p.valBuff) == 0 {
			p.Error(ErrNoValues.Args(p.cField, p.cStyle))
			return false
		}

		vals := make([]interface{}, len(p.valBuff))
		copy(vals, p.valBuff)
		p.parsed[p.cField] = vals
	}

	if !standalone {
		p.Error(ErrIncomplete)
	}
	return standalone
}



func (p *Parser) ident(tgt *string) bool {
	ident := p.Ident()
	if len(ident) == 0 {
		p.Error(ErrIdent)
		return false
	}
	if p.Ch != ':' {
		p.Error(ErrExpectedByte.Args(':', p.Ch))
		return false
	}
	*tgt = string(ident)
	return true
}*/
