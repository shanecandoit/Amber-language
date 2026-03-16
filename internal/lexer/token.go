// Package lexer tokenizes Amber source code.
package lexer

// TokenKind identifies a type of token.
type TokenKind int

const (
	// Literals
	TokNumber      TokenKind = iota
	TokString                // "..." or '...'
	TokTemplate              // `...`
	TokFingerprint           // #[0-9a-f]{64}
	TokTrue
	TokFalse
	TokUndefined

	// Keywords
	TokConst
	TokImport
	TokExport
	TokFrom
	TokAsync
	TokAwait
	TokMatch
	TokIf
	TokElse
	TokReturn
	TokSchema
	TokTable
	TokSpawn
	TokSend
	TokReceive
	TokFreeze
	TokThaw
	TokExit
	TokSelf

	// Identifiers
	TokIdent

	// Operators
	TokArrow    // =>
	TokSpread   // ...
	TokPlus     // +
	TokMinus    // -
	TokStar     // *
	TokSlash    // /
	TokPercent  // %
	TokEqEq     // ===
	TokNotEq    // !==
	TokLt       // <
	TokLtEq     // <=
	TokGt       // >
	TokGtEq     // >=
	TokAnd      // &&
	TokOr       // ||
	TokNot      // !
	TokEq       // =
	TokDot      // .
	TokComma    // ,
	TokColon    // :
	TokSemi     // ;
	TokQuestion // ?

	// Brackets
	TokLParen   // (
	TokRParen   // )
	TokLBrace   // {
	TokRBrace   // }
	TokLBracket // [
	TokRBracket // ]

	TokEOF
)

var kindNames = map[TokenKind]string{
	TokNumber:      "Number",
	TokString:      "String",
	TokTemplate:    "Template",
	TokFingerprint: "Fingerprint",
	TokTrue:        "true",
	TokFalse:       "false",
	TokUndefined:   "undefined",
	TokConst:       "const",
	TokImport:      "import",
	TokExport:      "export",
	TokFrom:        "from",
	TokAsync:       "async",
	TokAwait:       "await",
	TokMatch:       "match",
	TokIf:          "if",
	TokElse:        "else",
	TokReturn:      "return",
	TokSchema:      "schema",
	TokTable:       "table",
	TokSpawn:       "spawn",
	TokSend:        "send",
	TokReceive:     "receive",
	TokFreeze:      "freeze",
	TokThaw:        "thaw",
	TokExit:        "exit",
	TokSelf:        "self",
	TokIdent:       "Ident",
	TokArrow:       "=>",
	TokSpread:      "...",
	TokPlus:        "+",
	TokMinus:       "-",
	TokStar:        "*",
	TokSlash:       "/",
	TokPercent:     "%",
	TokEqEq:        "===",
	TokNotEq:       "!==",
	TokLt:          "<",
	TokLtEq:        "<=",
	TokGt:          ">",
	TokGtEq:        ">=",
	TokAnd:         "&&",
	TokOr:          "||",
	TokNot:         "!",
	TokEq:          "=",
	TokDot:         ".",
	TokComma:       ",",
	TokColon:       ":",
	TokSemi:        ";",
	TokQuestion:    "?",
	TokLParen:      "(",
	TokRParen:      ")",
	TokLBrace:      "{",
	TokRBrace:      "}",
	TokLBracket:    "[",
	TokRBracket:    "]",
	TokEOF:         "EOF",
}

func (k TokenKind) String() string {
	if s, ok := kindNames[k]; ok {
		return s
	}
	return "Unknown"
}

// Token is a single lexical token.
type Token struct {
	Kind  TokenKind
	Value string // raw source text of the token
	Line  int
	Col   int
}
