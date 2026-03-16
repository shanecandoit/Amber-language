// Command amber — the Amber language runner.
//
// Usage:
//
//	amber <file.amber>    run a source file
//	amber repl            start an interactive REPL
//	amber version         print version
package main

import (
	"fmt"
	"os"

	"github.com/shanecandoit/Amber-language/internal/lexer"
)

const version = "0.1.0-dev"

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Printf("amber %s\n", version)

	case "repl":
		fmt.Println("Amber REPL (not yet implemented)")
		fmt.Printf("amber %s\n", version)

	case "lex":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: amber lex <file.amber>")
			os.Exit(1)
		}
		runLex(args[1])

	default:
		if len(args[0]) > 0 && args[0][0] != '-' {
			// Treat as a source file
			fmt.Fprintf(os.Stderr, "evaluator not yet implemented — try: amber lex %s\n", args[0])
			os.Exit(1)
		}
		printUsage()
		os.Exit(1)
	}
}

func runLex(path string) {
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", path, err)
		os.Exit(1)
	}

	l := lexer.New(string(src))
	tokens, err := l.Tokenize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lex error: %v\n", err)
		os.Exit(1)
	}

	for _, tok := range tokens {
		fmt.Printf("%3d:%-3d  %-16s  %q\n", tok.Line, tok.Col, tok.Kind, tok.Value)
	}
}

func printUsage() {
	fmt.Println(`usage: amber <command> [args]

commands:
  <file.amber>    run a source file (not yet implemented)
  lex <file>      lex a source file and print tokens
  repl            start the interactive REPL (not yet implemented)
  version         print version`)
}
