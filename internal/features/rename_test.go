package features

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Test validation functions that don't require document parsing

func TestRenameProviderValidateNewName(t *testing.T) {
	provider := NewRenameProvider()

	testCases := []struct {
		name      string
		newName   string
		expectErr bool
	}{
		{"valid identifier", "NewUser", false},
		{"valid with underscore", "New_User", false},
		{"valid with number", "User2", false},
		{"empty string", "", true},
		{"whitespace only", "  ", true},
		{"starts with number", "2User", true},
		{"contains space", "New User", true},
		{"reserved keyword", "struct", true},
		{"built-in type", "string", true},
		{"special characters", "User@Name", true},
		{"hyphen", "User-Name", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := provider.validateNewName(tc.newName)
			if tc.expectErr && err == nil {
				t.Errorf("Expected error for name '%s' but got none", tc.newName)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error for name '%s' but got: %v", tc.newName, err)
			}
		})
	}
}

func TestRenameProviderIsRenameable(t *testing.T) {
	provider := NewRenameProvider()

	testCases := []struct {
		name      string
		symbol    *SymbolInfo
		renameable bool
	}{
		{
			name: "user-defined struct",
			symbol: &SymbolInfo{Name: "User", Kind: "struct"},
			renameable: true,
		},
		{
			name: "built-in type string",
			symbol: &SymbolInfo{Name: "string", Kind: "type"},
			renameable: false,
		},
		{
			name: "built-in type i32",
			symbol: &SymbolInfo{Name: "i32", Kind: "type"},
			renameable: false,
		},
		{
			name: "keyword struct",
			symbol: &SymbolInfo{Name: "struct", Kind: "keyword"},
			renameable: false,
		},
		{
			name: "keyword service",
			symbol: &SymbolInfo{Name: "service", Kind: "keyword"},
			renameable: false,
		},
		{
			name: "user field",
			symbol: &SymbolInfo{Name: "userId", Kind: "field"},
			renameable: true,
		},
		{
			name: "user method",
			symbol: &SymbolInfo{Name: "getUser", Kind: "method"},
			renameable: true,
		},
		{
			name: "user constant",
			symbol: &SymbolInfo{Name: "MAX_SIZE", Kind: "constant"},
			renameable: true,
		},
		{
			name: "nil symbol",
			symbol: nil,
			renameable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := provider.isRenameable(tc.symbol)
			if result != tc.renameable {
				t.Errorf("Expected renameable=%v for %v, got %v", tc.renameable, tc.symbol, result)
			}
		})
	}
}

func TestRenameProviderCheckConflicts(t *testing.T) {
	provider := NewRenameProvider()
	
	// Test basic conflict detection (same name)
	symbol := &SymbolInfo{Name: "User", Kind: "struct"}
	err := provider.checkConflicts(symbol, "User", nil)
	if err == nil {
		t.Error("Expected error when renaming to same name")
	}
	
	// Test valid rename (different name)
	err = provider.checkConflicts(symbol, "Person", nil)
	if err != nil {
		t.Errorf("Expected no error for valid rename, got: %v", err)
	}
}

func TestSymbolInfoCreation(t *testing.T) {
	// Test creating SymbolInfo with different properties
	testCases := []struct {
		name    string
		symbol  SymbolInfo
		valid   bool
	}{
		{
			name: "struct symbol",
			symbol: SymbolInfo{
				Name:    "User",
				Kind:    "struct",
				Context: "definition",
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 7},
					End:   protocol.Position{Line: 0, Character: 11},
				},
			},
			valid: true,
		},
		{
			name: "field symbol",
			symbol: SymbolInfo{
				Name:    "name",
				Kind:    "field",
				Context: "field",
				Range: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 14},
					End:   protocol.Position{Line: 1, Character: 18},
				},
			},
			valid: true,
		},
		{
			name: "empty name",
			symbol: SymbolInfo{
				Name:    "",
				Kind:    "struct",
				Context: "definition",
			},
			valid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.valid && tc.symbol.Name == "" {
				t.Error("Expected valid symbol to have non-empty name")
			}
			if !tc.valid && tc.symbol.Name != "" {
				// This test case expects an invalid symbol, so name should be empty
			}
		})
	}
}

func TestRenameProviderCreation(t *testing.T) {
	provider := NewRenameProvider()
	if provider == nil {
		t.Error("NewRenameProvider should return a valid provider")
	}
	
	if provider.referencesProvider == nil {
		t.Error("RenameProvider should have a references provider")
	}
}

func TestValidationEdgeCases(t *testing.T) {
	provider := NewRenameProvider()
	
	// Test edge cases for name validation
	testCases := []struct {
		name    string
		newName string
		valid   bool
	}{
		{"underscore only", "_", true},
		{"underscore start", "_user", true},  
		{"single letter", "a", true},
		{"all caps", "USER", true},
		{"mixed case", "UsErNaMe", true},
		{"numbers at end", "user123", true},
		{"unicode character", "usér", false}, // Not a valid ASCII identifier
		{"tab character", "user\t", false},
		{"newline character", "user\n", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := provider.validateNewName(tc.newName)
			if tc.valid && err != nil {
				t.Errorf("Expected '%s' to be valid, got error: %v", tc.newName, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("Expected '%s' to be invalid, but got no error", tc.newName)
			}
		})
	}
}