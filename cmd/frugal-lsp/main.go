package main

import (
	"fmt"
	"log"
	"os"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	"frugal-lsp/internal/parser"
	"frugal-lsp/pkg/ast"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: frugal-lsp <frugal-file>")
		fmt.Println("Running test with sample.frugal...")
		testParser()
		return
	}

	filename := os.Args[1]
	source, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", filename, err)
	}

	testParseFile(source, filename)
}

func testParser() {
	// Test with the sample frugal file
	samplePath := "tree-sitter-frugal/examples/sample.frugal"
	source, err := os.ReadFile(samplePath)
	if err != nil {
		log.Fatalf("Failed to read sample file: %v", err)
	}

	testParseFile(source, samplePath)
}

func testParseFile(source []byte, filename string) {
	fmt.Printf("Testing parser with file: %s\n", filename)
	fmt.Printf("Source length: %d bytes\n\n", len(source))

	// Create parser
	p, err := parser.NewParser()
	if err != nil {
		log.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	// Parse the source
	result, err := p.Parse(source)
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
		printLimitedTree(root, source, 0, 3)
		
		// Extract symbols
		fmt.Println("\nExtracted Symbols:")
		symbols := ast.ExtractSymbols(root, source)
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