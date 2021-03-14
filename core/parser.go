package core

import (
	"reflect"
	"testing"

	"github.com/jakubDoka/gogen/str"
	"github.com/jakubDoka/sterr"
)

// ErrReport reports error location
var ErrReport = sterr.New("located at %d:%d")

// Parser serves base for a parser
type Parser struct {
	Source             []byte
	I, Line, LineStart int
	Ch                 byte
	Err                error
}

// Restart restarts parser state for another parsing
func (p *Parser) Restart(Source []byte) {
	p.Source = Source
	p.I = -1
	p.Line = 0
	p.LineStart = 0
	p.Err = nil
}

// Failed returns whether error happened
func (p *Parser) Failed() bool {
	return p.Err != nil
}

// error sets p.Err and adds the Line info
func (p *Parser) Error(err sterr.Err) {
	p.Err = err.Wrap(p.ReportError())
}

// ReportError returns error with position data
func (p *Parser) ReportError() sterr.Err {
	return ErrReport.Args(p.Line, p.I-p.LineStart)
}

// CheckSlice checks if next len(lice) bytes are equal to slice content
//
// ok will be false if there is not enough bytes in p.Source
// equal will be true if slices are equal
func (p *Parser) CheckSlice(slice []byte) (equal, ok bool) {
	if len(p.Source) <= len(slice)+p.I {
		return
	}

	for i := 0; i < len(slice); i++ {
		if p.Source[p.I+i] != slice[i] {
			return false, true
		}
	}

	return true, true
}

// Check returns false if p.Advance succeeds and p.Ch == b
// othervise it rises error
func (p *Parser) Check(b byte, err sterr.Err) bool {
	ok := p.Advance() && p.Ch == b
	if !ok {
		p.Error(err)
	}
	return !ok
}

// SkipSpace ignores all invisible characters until it finds visible one
// if there is new Line character, it updates the p.Line and p.LineStart
//
// returns false if Advance fails
func (p *Parser) SkipSpace() bool {
	for p.Advance() {
		switch p.Ch {
		case ' ', '\t':
		case '\n':
			p.Line++
			p.LineStart = p.I + 1
		default:
			return true
		}
	}

	return false
}

// Number returns slice containing number, it returns empty slice if no number is present
// it assumes that curent byte is the beginning of number
func (p *Parser) Number() []byte {
	start := p.I
	for p.Advance() && IsNum(p.Ch) {
	}

	// to prevent too long offset, just '-' is not a number
	if p.Source[start] == '-' && p.Source[start+1] == p.Ch {
		p.I--
		p.Ch = '-'
		return nil
	}

	return p.Source[start:p.I]
}

// IsNum returns whether byte is a number
func IsNum(b byte) bool {
	return b == '.' || b >= '0' && b <= '9'
}

// IsNumStart also checks for negative number
func IsNumStart(b byte) bool {
	return IsNum(b) || b == '-'
}

// Ident reads Ident and returns slice where it is located
func (p *Parser) Ident() []byte {
	start := p.I
	if !str.IsIdent(p.Ch) {
		return nil
	}

	for p.Advance() && str.IsIdent(p.Ch) {
	}

	return p.Source[start:p.I]
}

// AdvanceOr raises error if advancement fails, return value of p.Advance is inverted
func (p *Parser) AdvanceOr(err sterr.Err) bool {
	ok := p.Advance()
	if !ok {
		p.Error(err)
	}
	return !ok
}

// Set sets the cursor
func (p *Parser) Set(idx int) {
	p.I = idx
	p.Ch = p.Source[p.I]
}

// Advance calls p.Peek acd increases i
func (p *Parser) Advance() bool {
	ok := p.Peek()
	if ok {
		p.I++
	}
	return ok
}

// Peek stores next byte in p.Ch, returns true is action wos successfull
func (p *Parser) Peek() bool {
	if p.I+1 >= len(p.Source) {
		return false
	}
	p.Ch = p.Source[p.I+1]
	return true
}

// Degrade goes one byte back, opposite of p.Advance
func (p *Parser) Degrade() bool {
	p.I--
	if p.I < 0 {
		return false
	}
	p.Ch = p.Source[p.I]
	return true
}

// TestEqual test whether two values are equal and reports error to t with nice formatting
func TestEqual(t *testing.T, a, b interface{}) bool {
	equal := reflect.DeepEqual(a, b)
	if !equal {
		t.Errorf("\n%#v\n%#v", a, b)
	}
	return equal
}
