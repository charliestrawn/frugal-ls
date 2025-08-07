package main

import (
	"fmt"
	"log"
	"os"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	"frugal-lsp/internal/lsp"
	"frugal-lsp/internal/parser"
	"frugal-lsp/pkg/ast"
)

func main() {
	// Check command line arguments
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--test":
			// Run parser test mode
			if len(os.Args) > 2 {
				testParseFile(os.Args[2])
			} else {
				testParser()
			}
			return
		case "--help", "-h":
			printUsage()
			return
		case "--version", "-v":
			fmt.Printf("frugal-lsp version %s\n", "0.1.0")
			return
		}
	}

	// Default: Run as LSP server
	runLSPServer()
}

func printUsage() {
	fmt.Println("Frugal Language Server Protocol (LSP) implementation")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  frugal-lsp                 Run as LSP server (default)")
	fmt.Println("  frugal-lsp --test [file]   Test parser with file or sample")
	fmt.Println("  frugal-lsp --version       Show version information")
	fmt.Println("  frugal-lsp --help          Show this help message")
	fmt.Println()
	fmt.Println("LSP Mode:")
	fmt.Println("  The server communicates via stdin/stdout using the Language")
	fmt.Println("  Server Protocol. Use with LSP-compatible editors like VS Code,")
	fmt.Println("  Vim, Emacs, etc.")
	fmt.Println()
	fmt.Println("Test Mode:")
	fmt.Println("  --test                     Parse sample.frugal (if available)")
	fmt.Println("  --test <file>              Parse specific .frugal file")
}

func runLSPServer() {
	// Create and run the LSP server
	server, err := lsp.NewServer()
	if err != nil {
		log.Fatalf("Failed to create LSP server: %v", err)
	}

	if err := server.Run(); err != nil {
		log.Fatalf("LSP server error: %v", err)
	}
}

func testParser() {
	// Test with sample frugal file from the tree-sitter grammar examples
	// This is mainly for development/testing purposes
	testFiles := []string{
		"examples/sample.frugal",
		"tree-sitter-frugal/examples/sample.frugal",
		"../tree-sitter-frugal/examples/sample.frugal",
	}

	var source []byte
	var filename string
	var err error

	for _, file := range testFiles {
		source, err = os.ReadFile(file)
		if err == nil {
			filename = file
			break
		}
	}

	if err != nil {
		fmt.Println("No sample.frugal file found. Creating a minimal test...")
		source = []byte(`// Test Frugal file
namespace go test

service TestService {
    string echo(1: string message)
}

scope Events {
    MessageSent: string
}`)
		filename = "test.frugal"
	}

	testParseFile(filename, source)
}

func testParseFile(filename string, source ...[]byte) {
	var content []byte
	var err error

	if len(source) > 0 {
		content = source[0]
	} else {
		content, err = os.ReadFile(filename)
		if err != nil {
			log.Fatalf("Failed to read file %s: %v", filename, err)
		}
	}

	fmt.Printf("Testing parser with file: %s\n", filename)
	fmt.Printf("Source length: %d bytes\n\n", len(content))

	// Create parser
	p, err := parser.NewParser()
	if err != nil {
		log.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	// Parse the source
	result, err := p.Parse(content)
	if err != nil {
		log.Fatalf("Failed to parse source: %v", err)
	}
	defer result.Close()

	fmt.Printf("Parse completed. Errors: %d\n", len(result.Errors))

	// Print any errors
	if result.HasErrors() {
		fmt.Println("\nParse Errors:")
		for _, err := range result.Errors {
			fmt.Printf("  Line %d, Column %d: %s\n", err.Line+1, err.Column+1, err.Message)
		}
		fmt.Println()
	} else {
		fmt.Println("✓ No parse errors found!")
	}

	// Print the AST tree (first few levels)
	root := result.GetRootNode()
	if root != nil {
		fmt.Println("\nAST Structure (first 3 levels):")
		printLimitedTree(root, content, 0, 3)
		
		// Extract symbols
		fmt.Println("\nExtracted Symbols:")
		symbols := ast.ExtractSymbols(root, content)
		if len(symbols) == 0 {
			fmt.Println("  No symbols found")
		} else {
			for _, symbol := range symbols {
				fmt.Printf("  %s %s (line %d, col %d)\n", 
					symbol.Type, symbol.Name, symbol.Line+1, symbol.Column+1)
			}
		}
	}

	fmt.Printf("\nParsing test completed for %s\n", filename)
}

func printLimitedTree(node *tree_sitter.Node, source []byte, indent int, maxDepth int) {
	if node == nil || indent > maxDepth {
		return
	}

	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += "  "
	}

	nodeText := ast.GetText(node, source)
	if len(nodeText) > 30 {
		nodeText = nodeText[:30] + "..."
	}
	nodeText = fmt.Sprintf("%q", nodeText)
	if nodeText == `""` {
		nodeText = "(empty)"
	}

	fmt.Printf("%s%s: %s\n", indentStr, node.Kind(), nodeText)

	if indent < maxDepth {
		childCount := node.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := node.Child(i)
			printLimitedTree(child, source, indent+1, maxDepth)
		}
	}
}