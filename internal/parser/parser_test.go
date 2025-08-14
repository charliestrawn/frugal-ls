package parser

import (
	"testing"
)

//nolint:gocognit // Test functions are naturally complex
func TestScopeDeclarations(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "scope with quoted prefix",
			source: `scope UserEvents prefix "user" {
    UserCreated: User,
    UserUpdated: User
}`,
			wantErr: false,
		},
		{
			name: "scope with unquoted prefix",
			source: `scope AlbumWinners prefix v1.music {
    AlbumWon: Album
}`,
			wantErr: false,
		},
		{
			name: "scope with dotted unquoted prefix",
			source: `scope NotificationEvents prefix v2.notifications.user {
    NotificationSent: Notification
}`,
			wantErr: false,
		},
		{
			name:    "empty scope",
			source:  `scope EmptyScope prefix "empty" {}`,
			wantErr: false,
		},
		{
			name: "scope with multiple events",
			source: `scope MultiEvents prefix "multi" {
    EventOne: TypeOne,
    EventTwo: TypeTwo,
    EventThree: TypeThree
}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}
			defer parser.Close()

			result, err := parser.Parse([]byte(tt.source))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			defer result.Close()

			hasErrors := result.HasErrors()
			if hasErrors != tt.wantErr {
				t.Errorf("expected hasErrors=%v, got=%v", tt.wantErr, hasErrors)
				if hasErrors {
					for _, parseErr := range result.Errors {
						t.Logf("Parse error: %s at line %d, col %d", parseErr.Message, parseErr.Line+1, parseErr.Column+1)
					}
				}
			}

			// Additional check: verify root node exists and has children
			root := result.GetRootNode()
			if root == nil {
				t.Fatal("root node is nil")
			}

			if root.ChildCount() == 0 {
				t.Error("root node has no children - parsing may have failed")
			}
		})
	}
}

func TestComplexFrugalFile(t *testing.T) {
	source := `include "common.frugal"

namespace go example

service UserService {
    User getUser(1: i64 userId) throws (1: UserNotFound error),
    void updateUser(1: User user)
}

scope UserEvents prefix "user" {
    UserCreated: User,
    UserUpdated: User
}

scope AlbumWinners prefix v1.music {
    AlbumWon: Album,
    AlbumNominated: Album
}

struct User {
    1: required i64 id,
    2: optional string name,
    3: optional string email
}

struct Album {
    1: required string title,
    2: required string artist,
    3: optional i32 year
}

exception UserNotFound {
    1: string message
}

const i32 DEFAULT_TIMEOUT = 5000;

enum Status {
    ACTIVE = 1,
    INACTIVE = 2,
    PENDING = 3
}

typedef string UserId
typedef map<string, string> Metadata`

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	result, err := parser.Parse([]byte(source))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	defer result.Close()

	if result.HasErrors() {
		t.Error("expected no parsing errors, but got:")
		for _, parseErr := range result.Errors {
			t.Errorf("  Parse error: %s at line %d, col %d", parseErr.Message, parseErr.Line+1, parseErr.Column+1)
		}
	}

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("root node is nil")
	}

	if root.ChildCount() == 0 {
		t.Error("root node has no children - parsing may have failed")
	}
}

func TestParserErrorCollection(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		expectErrs int
	}{
		{
			name:       "valid syntax",
			source:     `struct User { 1: string name }`,
			expectErrs: 0,
		},
		{
			name:       "invalid syntax - missing brace",
			source:     `struct User { 1: string name`,
			expectErrs: 1,
		},
		{
			name:       "invalid syntax - bad scope",
			source:     `scope BadScope prefix { invalid }`,
			expectErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}
			defer parser.Close()

			result, err := parser.Parse([]byte(tt.source))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			defer result.Close()

			errCount := len(result.Errors)
			if errCount < tt.expectErrs {
				t.Errorf("expected at least %d errors, got %d", tt.expectErrs, errCount)
			}

			if tt.expectErrs > 0 && !result.HasErrors() {
				t.Error("expected parsing errors but got none")
			}
		})
	}
}

func TestServiceDefinitions(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "simple service",
			source: `service UserService {
    User getUser(1: i64 userId)
}`,
			wantErr: false,
		},
		{
			name: "service with void method",
			source: `service UserService {
    void updateUser(1: User user)
}`,
			wantErr: false,
		},
		{
			name: "service with throws clause",
			source: `service UserService {
    User getUser(1: i64 userId) throws (1: UserNotFound error)
}`,
			wantErr: false,
		},
		{
			name: "service with inheritance",
			source: `service ExtendedService extends BaseService {
    void newMethod()
}`,
			wantErr: false,
		},
		{
			name: "service with oneway method",
			source: `service NotificationService {
    oneway void sendNotification(1: string message)
}`,
			wantErr: false,
		},
		{
			name: "service with annotations",
			source: `service AnnotatedService {
    string getValue() (deprecated="use getNewValue instead")
}`,
			wantErr: false,
		},
		{
			name: "service with multiple methods",
			source: `service MultiService {
    User getUser(1: i64 id),
    void updateUser(1: User user),
    list<User> getAllUsers()
}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}
			defer parser.Close()

			result, err := parser.Parse([]byte(tt.source))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			defer result.Close()

			hasErrors := result.HasErrors()
			if hasErrors != tt.wantErr {
				t.Errorf("expected hasErrors=%v, got=%v", tt.wantErr, hasErrors)
				if hasErrors {
					for _, parseErr := range result.Errors {
						t.Logf("Parse error: %s at line %d, col %d", parseErr.Message, parseErr.Line+1, parseErr.Column+1)
					}
				}
			}
		})
	}
}

func TestStructAndExceptionDefinitions(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "simple struct",
			source: `struct User {
    1: string name,
    2: i64 id
}`,
			wantErr: false,
		},
		{
			name: "struct with required/optional fields",
			source: `struct User {
    1: required i64 id,
    2: optional string name,
    3: optional string email
}`,
			wantErr: false,
		},
		{
			name: "struct with default values",
			source: `struct Config {
    1: i32 timeout = 5000,
    2: string host = "localhost",
    3: bool enabled = true
}`,
			wantErr: false,
		},
		{
			name: "exception definition",
			source: `exception UserNotFound {
    1: string message,
    2: i64 userId
}`,
			wantErr: false,
		},
		{
			name: "union definition",
			source: `union Response {
    1: User user,
    2: string error
}`,
			wantErr: false,
		},
		{
			name: "struct with annotations",
			source: `struct AnnotatedStruct {
    1: string field (deprecated="use newField")
} (table="users")`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}
			defer parser.Close()

			result, err := parser.Parse([]byte(tt.source))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			defer result.Close()

			hasErrors := result.HasErrors()
			if hasErrors != tt.wantErr {
				t.Errorf("expected hasErrors=%v, got=%v", tt.wantErr, hasErrors)
				if hasErrors {
					for _, parseErr := range result.Errors {
						t.Logf("Parse error: %s at line %d, col %d", parseErr.Message, parseErr.Line+1, parseErr.Column+1)
					}
				}
			}
		})
	}
}

func TestComplexTypes(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "list types",
			source: `struct Container {
    1: list<string> names,
    2: list<i64> numbers,
    3: list<User> users
}`,
			wantErr: false,
		},
		{
			name: "set types",
			source: `struct Container {
    1: set<string> tags,
    2: set<i32> ids
}`,
			wantErr: false,
		},
		{
			name: "map types",
			source: `struct Container {
    1: map<string, string> metadata,
    2: map<i64, User> userMap,
    3: map<string, list<string>> groupedData
}`,
			wantErr: false,
		},
		{
			name: "nested container types",
			source: `struct Complex {
    1: list<map<string, set<i32>>> nestedData,
    2: map<string, list<User>> userGroups
}`,
			wantErr: false,
		},
		{
			name: "all base types",
			source: `struct AllTypes {
    1: bool flag,
    2: byte b,
    3: i8 tiny,
    4: i16 small,
    5: i32 medium,
    6: i64 large,
    7: double precise,
    8: string text,
    9: binary data
}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}
			defer parser.Close()

			result, err := parser.Parse([]byte(tt.source))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			defer result.Close()

			hasErrors := result.HasErrors()
			if hasErrors != tt.wantErr {
				t.Errorf("expected hasErrors=%v, got=%v", tt.wantErr, hasErrors)
				if hasErrors {
					for _, parseErr := range result.Errors {
						t.Logf("Parse error: %s at line %d, col %d", parseErr.Message, parseErr.Line+1, parseErr.Column+1)
					}
				}
			}
		})
	}
}

func TestNamespaceAndIncludes(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "include directive",
			source:  `include "common.frugal"`,
			wantErr: false,
		},
		{
			name:    "include with single quotes",
			source:  `include 'shared.frugal'`,
			wantErr: false,
		},
		{
			name:    "namespace go",
			source:  `namespace go example.service`,
			wantErr: false,
		},
		{
			name:    "namespace java",
			source:  `namespace java com.example`,
			wantErr: false,
		},
		{
			name:    "namespace python",
			source:  `namespace python example.service`,
			wantErr: false,
		},
		{
			name:    "namespace cpp",
			source:  `namespace cpp example`,
			wantErr: false,
		},
		{
			name:    "namespace wildcard",
			source:  `namespace * example`,
			wantErr: false,
		},
		{
			name: "multiple includes and namespace",
			source: `include "common.frugal"
include "types.frugal"
namespace go example`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}
			defer parser.Close()

			result, err := parser.Parse([]byte(tt.source))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			defer result.Close()

			hasErrors := result.HasErrors()
			if hasErrors != tt.wantErr {
				t.Errorf("expected hasErrors=%v, got=%v", tt.wantErr, hasErrors)
				if hasErrors {
					for _, parseErr := range result.Errors {
						t.Logf("Parse error: %s at line %d, col %d", parseErr.Message, parseErr.Line+1, parseErr.Column+1)
					}
				}
			}
		})
	}
}

func TestEnumsAndConstants(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "simple enum",
			source: `enum Status {
    ACTIVE,
    INACTIVE,
    PENDING
}`,
			wantErr: false,
		},
		{
			name: "enum with values",
			source: `enum Status {
    ACTIVE = 1,
    INACTIVE = 2,
    PENDING = 3
}`,
			wantErr: false,
		},
		{
			name: "const definitions",
			source: `const i32 DEFAULT_TIMEOUT = 5000;
const string API_VERSION = "v1.0";
const bool FEATURE_ENABLED = true;`,
			wantErr: false,
		},
		{
			name: "const with complex values",
			source: `const list<string> ALLOWED_METHODS = ["GET", "POST", "PUT"];
const map<string, i32> ERROR_CODES = {"NOT_FOUND": 404, "SERVER_ERROR": 500};`,
			wantErr: false,
		},
		{
			name: "typedef definitions",
			source: `typedef string UserId
typedef map<string, string> Metadata
typedef list<User> UserList`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}
			defer parser.Close()

			result, err := parser.Parse([]byte(tt.source))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			defer result.Close()

			hasErrors := result.HasErrors()
			if hasErrors != tt.wantErr {
				t.Errorf("expected hasErrors=%v, got=%v", tt.wantErr, hasErrors)
				if hasErrors {
					for _, parseErr := range result.Errors {
						t.Logf("Parse error: %s at line %d, col %d", parseErr.Message, parseErr.Line+1, parseErr.Column+1)
					}
				}
			}
		})
	}
}

func TestCommentsAndAnnotations(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "line comments",
			source: `// This is a comment
struct User {
    1: string name // field comment
}`,
			wantErr: false,
		},
		{
			name: "block comments",
			source: `/* Block comment
   spanning multiple lines */
struct User {
    1: string name
}`,
			wantErr: false,
		},
		{
			name: "hash comments",
			source: `# Hash-style comment
struct User {
    1: string name
}`,
			wantErr: true, // Hash comments may not be supported
		},
		{
			name: "struct with annotations",
			source: `struct User {
    1: string name (required="true")
} (table="users", deprecated="use UserV2")`,
			wantErr: false,
		},
		{
			name: "service with annotations",
			source: `service UserService {
    User getUser(1: i64 id) (timeout="5s")
} (version="1.0")`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}
			defer parser.Close()

			result, err := parser.Parse([]byte(tt.source))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			defer result.Close()

			hasErrors := result.HasErrors()
			if hasErrors != tt.wantErr {
				t.Errorf("expected hasErrors=%v, got=%v", tt.wantErr, hasErrors)
				if hasErrors {
					for _, parseErr := range result.Errors {
						t.Logf("Parse error: %s at line %d, col %d", parseErr.Message, parseErr.Line+1, parseErr.Column+1)
					}
				}
			}
		})
	}
}

func TestEdgeCasesAndErrors(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		expectErrs int
	}{
		{
			name:       "empty file",
			source:     "",
			expectErrs: 0,
		},
		{
			name:       "whitespace only",
			source:     "   \n\t  \n",
			expectErrs: 0,
		},
		{
			name:       "comments only",
			source:     "// Just comments\n/* Nothing else */",
			expectErrs: 0,
		},
		{
			name:       "missing semicolon in struct",
			source:     "struct User { 1: string name }",
			expectErrs: 0, // semicolons are optional
		},
		{
			name:       "missing field ID",
			source:     "struct User { string name }",
			expectErrs: 0, // field IDs are optional
		},
		{
			name:       "unclosed struct brace",
			source:     "struct User { 1: string name",
			expectErrs: 1,
		},
		{
			name:       "invalid field ID",
			source:     "struct User { abc: string name }",
			expectErrs: 1,
		},
		{
			name:       "malformed service",
			source:     "service { void method() }",
			expectErrs: 1,
		},
		{
			name:       "invalid scope prefix",
			source:     "scope Test prefix { }",
			expectErrs: 1,
		},
		{
			name:       "duplicate field IDs",
			source:     "struct User { 1: string name, 1: string email }",
			expectErrs: 0, // grammar allows this, semantic validation would catch it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}
			defer parser.Close()

			result, err := parser.Parse([]byte(tt.source))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			defer result.Close()

			errCount := len(result.Errors)
			if tt.expectErrs == 0 && errCount > 0 {
				t.Errorf("expected no errors, got %d", errCount)
				for _, parseErr := range result.Errors {
					t.Logf("Parse error: %s at line %d, col %d", parseErr.Message, parseErr.Line+1, parseErr.Column+1)
				}
			} else if tt.expectErrs > 0 && errCount < tt.expectErrs {
				t.Errorf("expected at least %d errors, got %d", tt.expectErrs, errCount)
			}
		})
	}
}

func TestParseResultMethods(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	source := `struct Test { 1: string field }`
	result, err := parser.Parse([]byte(source))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	defer result.Close()

	// Test GetRootNode
	root := result.GetRootNode()
	if root == nil {
		t.Error("GetRootNode returned nil")
	}

	// Test HasErrors for valid syntax
	if result.HasErrors() {
		t.Error("HasErrors returned true for valid syntax")
	}

	// Test with invalid syntax
	invalidResult, err := parser.Parse([]byte(`invalid syntax {`))
	if err != nil {
		t.Fatalf("failed to parse invalid syntax: %v", err)
	}
	defer invalidResult.Close()

	if !invalidResult.HasErrors() {
		t.Error("HasErrors returned false for invalid syntax")
	}
}
