package lexer

import (
	"fmt"
	"strings"
	"unicode"
)

var keywords = map[string]TokenKind{
	"true":      TokTrue,
	"false":     TokFalse,
	"undefined": TokUndefined,
	"const":     TokConst,
	"import":    TokImport,
	"export":    TokExport,
	"from":      TokFrom,
	"async":     TokAsync,
	"await":     TokAwait,
	"match":     TokMatch,
	"if":        TokIf,
	"else":      TokElse,
	"return":    TokReturn,
	"schema":    TokSchema,
	"table":     TokTable,
	"spawn":     TokSpawn,
	"send":      TokSend,
	"receive":   TokReceive,
	"freeze":    TokFreeze,
	"thaw":      TokThaw,
	"exit":      TokExit,
	"self":      TokSelf,
}

// Lexer tokenizes Amber source code.
type Lexer struct {
	src    []rune
	pos    int
	line   int
	col    int
	tokens []Token
}

// New creates a new Lexer for the given source.
func New(src string) *Lexer {
	return &Lexer{src: []rune(src), line: 1, col: 1}
}

// Tokenize lexes the entire source and returns all tokens.
func (l *Lexer) Tokenize() ([]Token, error) {
	for {
		tok, err := l.next()
		if err != nil {
			return nil, err
		}
		l.tokens = append(l.tokens, tok)
		if tok.Kind == TokEOF {
			break
		}
	}
	return l.tokens, nil
}

func (l *Lexer) next() (Token, error) {
	l.skipWhitespaceAndComments()

	if l.pos >= len(l.src) {
		return l.tok(TokEOF, ""), nil
	}

	startLine, startCol := l.line, l.col
	ch := l.src[l.pos]

	// Fingerprint literal: #[0-9a-f]{64}
	if ch == '#' && l.pos+65 <= len(l.src) {
		candidate := string(l.src[l.pos+1 : l.pos+65])
		if isHexString(candidate) {
			raw := "#" + candidate
			l.advance(65)
			return Token{Kind: TokFingerprint, Value: raw, Line: startLine, Col: startCol}, nil
		}
	}

	// String literals
	if ch == '"' || ch == '\'' {
		return l.readString(ch, startLine, startCol)
	}

	// Template literals
	if ch == '`' {
		return l.readTemplate(startLine, startCol)
	}

	// Numbers
	if unicode.IsDigit(ch) || (ch == '-' && l.pos+1 < len(l.src) && unicode.IsDigit(l.src[l.pos+1])) {
		return l.readNumber(startLine, startCol)
	}

	// Identifiers and keywords
	if unicode.IsLetter(ch) || ch == '_' {
		return l.readIdent(startLine, startCol)
	}

	// Multi-char operators
	switch ch {
	case '.':
		if l.peek(1) == '.' && l.peek(2) == '.' {
			l.advance(3)
			return Token{Kind: TokSpread, Value: "...", Line: startLine, Col: startCol}, nil
		}
		l.advance(1)
		return Token{Kind: TokDot, Value: ".", Line: startLine, Col: startCol}, nil

	case '=':
		if l.peek(1) == '=' && l.peek(2) == '=' {
			l.advance(3)
			return Token{Kind: TokEqEq, Value: "===", Line: startLine, Col: startCol}, nil
		}
		if l.peek(1) == '>' {
			l.advance(2)
			return Token{Kind: TokArrow, Value: "=>", Line: startLine, Col: startCol}, nil
		}
		l.advance(1)
		return Token{Kind: TokEq, Value: "=", Line: startLine, Col: startCol}, nil

	case '!':
		if l.peek(1) == '=' && l.peek(2) == '=' {
			l.advance(3)
			return Token{Kind: TokNotEq, Value: "!==", Line: startLine, Col: startCol}, nil
		}
		l.advance(1)
		return Token{Kind: TokNot, Value: "!", Line: startLine, Col: startCol}, nil

	case '<':
		if l.peek(1) == '=' {
			l.advance(2)
			return Token{Kind: TokLtEq, Value: "<=", Line: startLine, Col: startCol}, nil
		}
		l.advance(1)
		return Token{Kind: TokLt, Value: "<", Line: startLine, Col: startCol}, nil

	case '>':
		if l.peek(1) == '=' {
			l.advance(2)
			return Token{Kind: TokGtEq, Value: ">=", Line: startLine, Col: startCol}, nil
		}
		l.advance(1)
		return Token{Kind: TokGt, Value: ">", Line: startLine, Col: startCol}, nil

	case '&':
		if l.peek(1) == '&' {
			l.advance(2)
			return Token{Kind: TokAnd, Value: "&&", Line: startLine, Col: startCol}, nil
		}

	case '|':
		if l.peek(1) == '|' {
			l.advance(2)
			return Token{Kind: TokOr, Value: "||", Line: startLine, Col: startCol}, nil
		}
	}

	// Single-char tokens
	single := map[rune]TokenKind{
		'+': TokPlus, '-': TokMinus, '*': TokStar, '/': TokSlash, '%': TokPercent,
		',': TokComma, ':': TokColon, ';': TokSemi, '?': TokQuestion,
		'(': TokLParen, ')': TokRParen,
		'{': TokLBrace, '}': TokRBrace,
		'[': TokLBracket, ']': TokRBracket,
	}
	if kind, ok := single[ch]; ok {
		l.advance(1)
		return Token{Kind: kind, Value: string(ch), Line: startLine, Col: startCol}, nil
	}

	return Token{}, fmt.Errorf("unexpected character %q at line %d col %d", ch, l.line, l.col)
}

func (l *Lexer) readString(quote rune, line, col int) (Token, error) {
	l.advance(1) // opening quote
	var sb strings.Builder
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == quote {
			l.advance(1)
			return Token{Kind: TokString, Value: sb.String(), Line: line, Col: col}, nil
		}
		if ch == '\\' && l.pos+1 < len(l.src) {
			l.advance(1)
			switch l.src[l.pos] {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case 'r':
				sb.WriteRune('\r')
			default:
				sb.WriteRune(l.src[l.pos])
			}
			l.advance(1)
			continue
		}
		sb.WriteRune(ch)
		l.advance(1)
	}
	return Token{}, fmt.Errorf("unterminated string at line %d col %d", line, col)
}

func (l *Lexer) readTemplate(line, col int) (Token, error) {
	l.advance(1) // opening backtick
	var sb strings.Builder
	sb.WriteRune('`')
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == '`' {
			sb.WriteRune('`')
			l.advance(1)
			return Token{Kind: TokTemplate, Value: sb.String(), Line: line, Col: col}, nil
		}
		sb.WriteRune(ch)
		l.advance(1)
	}
	return Token{}, fmt.Errorf("unterminated template literal at line %d col %d", line, col)
}

func (l *Lexer) readNumber(line, col int) (Token, error) {
	start := l.pos
	if l.src[l.pos] == '-' {
		l.advance(1)
	}
	for l.pos < len(l.src) && (unicode.IsDigit(l.src[l.pos]) || l.src[l.pos] == '.' || l.src[l.pos] == 'e' || l.src[l.pos] == 'E' || l.src[l.pos] == '_') {
		l.advance(1)
	}
	return Token{Kind: TokNumber, Value: string(l.src[start:l.pos]), Line: line, Col: col}, nil
}

func (l *Lexer) readIdent(line, col int) (Token, error) {
	start := l.pos
	for l.pos < len(l.src) && (unicode.IsLetter(l.src[l.pos]) || unicode.IsDigit(l.src[l.pos]) || l.src[l.pos] == '_') {
		l.advance(1)
	}
	word := string(l.src[start:l.pos])
	kind := TokIdent
	if k, ok := keywords[word]; ok {
		kind = k
	}
	return Token{Kind: kind, Value: word, Line: line, Col: col}, nil
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			l.advance(1)
			continue
		}
		// Line comment
		if ch == '/' && l.peek(1) == '/' {
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.advance(1)
			}
			continue
		}
		break
	}
}

func (l *Lexer) advance(n int) {
	for i := 0; i < n && l.pos < len(l.src); i++ {
		if l.src[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

func (l *Lexer) peek(offset int) rune {
	idx := l.pos + offset
	if idx >= len(l.src) {
		return 0
	}
	return l.src[idx]
}

func (l *Lexer) tok(kind TokenKind, value string) Token {
	return Token{Kind: kind, Value: value, Line: l.line, Col: l.col}
}

func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
