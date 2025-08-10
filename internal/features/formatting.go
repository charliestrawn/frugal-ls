package features

import (
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"frugal-ls/internal/document"
)

// FormattingProvider handles document formatting for Frugal files
type FormattingProvider struct{}

// NewFormattingProvider creates a new formatting provider
func NewFormattingProvider() *FormattingProvider {
	return &FormattingProvider{}
}

// ProvideDocumentFormatting formats an entire document
func (f *FormattingProvider) ProvideDocumentFormatting(doc *document.Document, options protocol.FormattingOptions) ([]protocol.TextEdit, error) {
	if !doc.IsValidFrugalFile() {
		return nil, nil
	}

	// Use conservative indentation-only formatting
	formattedContent := f.formatDocumentConservatively(doc.Content, options)

	// If content unchanged, return no edits
	if formattedContent == string(doc.Content) {
		return nil, nil
	}

	// Return a single edit that replaces the entire document
	lines := strings.Split(string(doc.Content), "\n")
	endLine := uint32(len(lines) - 1)
	endChar := uint32(0)
	if len(lines) > 0 {
		endChar = uint32(len(lines[len(lines)-1]))
	}

	return []protocol.TextEdit{{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: endLine, Character: endChar},
		},
		NewText: formattedContent,
	}}, nil
}

// ProvideDocumentRangeFormatting formats a range of the document
func (f *FormattingProvider) ProvideDocumentRangeFormatting(doc *document.Document, rng protocol.Range, options protocol.FormattingOptions) ([]protocol.TextEdit, error) {
	// For conservative formatting, just format the entire document
	// Range formatting with conservative approach would be complex to implement correctly
	return f.ProvideDocumentFormatting(doc, options)
}

// formatDocument applies formatting rules to the document
func (f *FormattingProvider) formatDocument(source []byte, root interface{}, options protocol.FormattingOptions) string {
	lines := strings.Split(string(source), "\n")
	var formattedLines []string

	indentLevel := 0
	indentString := f.getIndentString(options)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines
		if trimmedLine == "" {
			formattedLines = append(formattedLines, "")
			continue
		}

		// Adjust indent level based on content
		if f.isClosingBrace(trimmedLine) {
			indentLevel = max(0, indentLevel-1)
		}

		// Apply indentation
		formattedLine := strings.Repeat(indentString, indentLevel) + trimmedLine
		formattedLines = append(formattedLines, formattedLine)

		// Increase indent level for opening braces
		if f.isOpeningBrace(trimmedLine) {
			indentLevel++
		}
	}

	// Apply additional formatting rules
	formatted := strings.Join(formattedLines, "\n")
	formatted = f.normalizeSpacing(formatted)
	formatted = f.formatComments(formatted)

	return formatted
}

// getIndentString returns the indentation string based on formatting options
func (f *FormattingProvider) getIndentString(options protocol.FormattingOptions) string {
	// Check if insertSpaces is set to true
	if insertSpaces, ok := options["insertSpaces"].(bool); ok && insertSpaces {
		// Get tabSize, default to 4 if not set or invalid
		tabSize := 4
		if ts, ok := options["tabSize"].(int); ok && ts > 0 {
			tabSize = ts
		} else if ts, ok := options["tabSize"].(float64); ok && ts > 0 {
			tabSize = int(ts)
		}
		return strings.Repeat(" ", tabSize)
	}
	return "\t"
}

// isOpeningBrace checks if a line contains an opening brace that increases indentation
func (f *FormattingProvider) isOpeningBrace(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasSuffix(line, "{") ||
		strings.Contains(line, "service ") && strings.HasSuffix(line, "{") ||
		strings.Contains(line, "scope ") && strings.HasSuffix(line, "{") ||
		strings.Contains(line, "struct ") && strings.HasSuffix(line, "{") ||
		strings.Contains(line, "enum ") && strings.HasSuffix(line, "{") ||
		strings.Contains(line, "exception ") && strings.HasSuffix(line, "{")
}

// isClosingBrace checks if a line contains a closing brace that decreases indentation
func (f *FormattingProvider) isClosingBrace(line string) bool {
	line = strings.TrimSpace(line)
	return line == "}" || strings.HasPrefix(line, "}")
}

// formatComments ensures proper spacing around comments
func (f *FormattingProvider) formatComments(content string) string {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			// Ensure single space after //
			comment := strings.TrimPrefix(trimmed, "//")
			comment = strings.TrimLeft(comment, " ")
			indent := strings.Repeat(" ", len(line)-len(strings.TrimLeft(line, " \t")))
			lines[i] = indent + "// " + comment
		}
	}

	return strings.Join(lines, "\n")
}

// formatDocumentConservatively applies proper indentation and basic spacing normalization
func (f *FormattingProvider) formatDocumentConservatively(source []byte, options protocol.FormattingOptions) string {
	lines := strings.Split(string(source), "\n")
	var formattedLines []string

	indentLevel := 0
	indentString := f.getIndentString(options)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Preserve empty lines as-is
		if trimmedLine == "" {
			formattedLines = append(formattedLines, "")
			continue
		}

		// Handle comments - preserve the original comment formatting exactly
		if strings.HasPrefix(trimmedLine, "//") || strings.HasPrefix(trimmedLine, "/*") || strings.HasPrefix(trimmedLine, "*") || strings.HasPrefix(trimmedLine, "*/") {
			// For comments at indent level 0, preserve exactly as-is
			if indentLevel == 0 {
				formattedLines = append(formattedLines, line)
			} else {
				// For nested comments, add base indentation but preserve comment structure
				baseIndent := strings.Repeat(indentString, indentLevel)
				// Remove any existing indentation and add the correct base indentation
				commentContent := strings.TrimLeft(line, " \t")
				formattedLine := baseIndent + commentContent
				formattedLines = append(formattedLines, formattedLine)
			}
			continue
		}

		// Adjust indent level based on closing braces
		if f.isClosingBrace(trimmedLine) {
			indentLevel = max(0, indentLevel-1)
		}

		// Apply proper indentation and basic spacing normalization
		normalizedLine := f.normalizeSpacing(trimmedLine)
		formattedLine := strings.Repeat(indentString, indentLevel) + normalizedLine
		formattedLines = append(formattedLines, formattedLine)

		// Increase indent level for opening braces
		if f.isOpeningBrace(trimmedLine) {
			indentLevel++
		}
	}

	return strings.Join(formattedLines, "\n")
}

// normalizeSpacing normalizes spacing in a line (convert tabs to spaces, fix multiple spaces)
func (f *FormattingProvider) normalizeSpacing(line string) string {
	// Replace tabs with spaces
	line = strings.ReplaceAll(line, "\t", "    ")

	// Normalize spacing around operators and punctuation
	line = strings.ReplaceAll(line, ",", ", ")
	line = strings.ReplaceAll(line, ",  ", ", ") // Fix double spaces after comma

	// Fix spacing around colons (but be careful with URLs)
	if strings.Contains(line, ":") && !strings.Contains(line, "://") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			before := strings.TrimSpace(parts[0])
			after := strings.TrimSpace(parts[1])
			line = before + ": " + after
		}
	}

	// Normalize multiple spaces to single spaces first
	for strings.Contains(line, "  ") {
		line = strings.ReplaceAll(line, "  ", " ")
	}

	// Trim spaces inside parentheses for method parameter lists
	// Remove space after opening paren: "( " -> "("
	line = strings.ReplaceAll(line, "( ", "(")
	// Remove space before closing paren: " )" -> ")"
	line = strings.ReplaceAll(line, " )", ")")

	return line
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
