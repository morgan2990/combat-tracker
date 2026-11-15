package mpmb

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ParseLiteral reads the subset of a JavaScript object literal that MPMB
// stores in its "*.Stringified" fields: objects with bare or quoted keys,
// arrays, single- or double-quoted strings, numbers, booleans, null. It
// returns map[string]any, []any, string, float64, bool or nil.
//
// It is a data parser, not an evaluator: anything else — a function, an
// identifier, a call, an operator — is an error. Nothing is ever executed.
func ParseLiteral(src string) (any, error) {
	p := &literalParser{s: src}
	v, err := p.value(0)
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.pos != len(p.s) {
		return nil, p.errorf("unexpected trailing text")
	}
	return v, nil
}

const maxLiteralDepth = 64

type literalParser struct {
	s   string
	pos int
}

func (p *literalParser) errorf(format string, args ...any) error {
	return fmt.Errorf("literal at offset %d: %s", p.pos, fmt.Sprintf(format, args...))
}

func (p *literalParser) peek() byte {
	if p.pos >= len(p.s) {
		return 0
	}
	return p.s[p.pos]
}

func (p *literalParser) skipSpace() {
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			p.pos++
			continue
		}
		return
	}
}

func (p *literalParser) value(depth int) (any, error) {
	if depth > maxLiteralDepth {
		return nil, p.errorf("nested too deeply")
	}
	p.skipSpace()
	switch c := p.peek(); {
	case c == '(':
		// MPMB wraps whole literals in parentheses, and writes "undefined" as
		// (void 0). Grouping is data; only the grouped value matters.
		p.pos++
		v, err := p.value(depth + 1)
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if p.peek() != ')' {
			return nil, p.errorf("expected )")
		}
		p.pos++
		return v, nil
	case c == '{':
		return p.object(depth)
	case c == '[':
		return p.array(depth)
	case c == '"' || c == '\'':
		return p.str()
	case c == '-' || c == '+' || (c >= '0' && c <= '9') || c == '.':
		return p.number()
	case isIdentStart(c):
		start := p.pos
		id := p.ident()
		switch id {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "null", "undefined":
			return nil, nil
		case "void":
			// "void 0" is JavaScript's spelling of undefined; any other operand is not data.
			p.skipSpace()
			if p.peek() != '0' {
				p.pos = start
				return nil, p.errorf("unexpected operand after void")
			}
			p.pos++
			return nil, nil
		}
		p.pos = start
		return nil, p.errorf("unexpected identifier %q", id)
	case c == 0:
		return nil, p.errorf("unexpected end of input")
	default:
		return nil, p.errorf("unexpected character %q", c)
	}
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

func (p *literalParser) ident() string {
	start := p.pos
	for p.pos < len(p.s) && isIdentPart(p.s[p.pos]) {
		p.pos++
	}
	return p.s[start:p.pos]
}

func (p *literalParser) object(depth int) (any, error) {
	p.pos++ // {
	out := map[string]any{}
	for {
		p.skipSpace()
		if p.peek() == '}' {
			p.pos++
			return out, nil
		}
		var key string
		switch c := p.peek(); {
		case c == '"' || c == '\'':
			s, err := p.str()
			if err != nil {
				return nil, err
			}
			key = s.(string)
		case isIdentStart(c):
			key = p.ident()
		case c >= '0' && c <= '9':
			n, err := p.number()
			if err != nil {
				return nil, err
			}
			key = strconv.FormatFloat(n.(float64), 'f', -1, 64)
		default:
			return nil, p.errorf("expected a key")
		}
		p.skipSpace()
		if p.peek() != ':' {
			return nil, p.errorf("expected : after key %q", key)
		}
		p.pos++
		v, err := p.value(depth + 1)
		if err != nil {
			return nil, err
		}
		out[key] = v
		p.skipSpace()
		switch p.peek() {
		case ',':
			p.pos++
		case '}':
			// handled at the top of the loop
		default:
			return nil, p.errorf("expected , or } after value of %q", key)
		}
	}
}

func (p *literalParser) array(depth int) (any, error) {
	p.pos++ // [
	out := []any{}
	for {
		p.skipSpace()
		if p.peek() == ']' {
			p.pos++
			return out, nil
		}
		v, err := p.value(depth + 1)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
		p.skipSpace()
		switch p.peek() {
		case ',':
			p.pos++
		case ']':
		default:
			return nil, p.errorf("expected , or ] in array")
		}
	}
}

func (p *literalParser) number() (any, error) {
	start := p.pos
	if c := p.peek(); c == '-' || c == '+' {
		p.pos++
	}
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if (c >= '0' && c <= '9') || c == '.' || c == 'e' || c == 'E' || ((c == '-' || c == '+') && (p.s[p.pos-1] == 'e' || p.s[p.pos-1] == 'E')) {
			p.pos++
			continue
		}
		break
	}
	f, err := strconv.ParseFloat(p.s[start:p.pos], 64)
	if err != nil {
		p.pos = start
		return nil, p.errorf("invalid number")
	}
	return f, nil
}

func (p *literalParser) str() (any, error) {
	quote := p.s[p.pos]
	p.pos++
	var b strings.Builder
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		switch {
		case c == quote:
			p.pos++
			return b.String(), nil
		case c == '\\':
			p.pos++
			if p.pos >= len(p.s) {
				return nil, p.errorf("unterminated string")
			}
			e := p.s[p.pos]
			p.pos++
			switch e {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'v':
				b.WriteByte('\v')
			case '0':
				b.WriteByte(0)
			case 'x', 'u':
				n := 2
				if e == 'u' {
					n = 4
				}
				if p.pos+n > len(p.s) {
					return nil, p.errorf("invalid escape")
				}
				code, err := strconv.ParseUint(p.s[p.pos:p.pos+n], 16, 32)
				if err != nil {
					return nil, p.errorf("invalid escape")
				}
				p.pos += n
				b.WriteRune(rune(code))
			default:
				// \\ \' \" \/ and any other escaped character stand for themselves.
				b.WriteByte(e)
			}
		case c == '\n':
			return nil, p.errorf("newline in string")
		default:
			b.WriteByte(c)
			p.pos++
		}
	}
	return nil, p.errorf("unterminated string")
}

// --- accessors for parsed literals ---

// lit digs into a parsed literal along keys, returning nil if any step is missing.
func lit(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

func litString(v any) string {
	s, _ := v.(string)
	return s
}

func litInt(v any) (int, bool) {
	f, ok := v.(float64)
	if !ok {
		return 0, false
	}
	return int(f), true
}

// titleCase upper-cases the first letter of each word.
func titleCase(s string) string {
	prev := ' '
	return strings.Map(func(r rune) rune {
		defer func() { prev = r }()
		if prev == ' ' || prev == '(' || prev == '-' {
			return unicode.ToUpper(r)
		}
		return r
	}, s)
}
