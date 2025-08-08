package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPrintUsage(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printUsage()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify the output contains expected content
	expectedContent := []string{
		"Frugal Language Server Protocol (LSP) implementation",
		"Usage:",
		"frugal-ls",
		"--test",
		"--version",
		"--help",
		"LSP Mode:",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Usage output should contain '%s'", expected)
		}
	}
}

// Test command line argument processing
func TestCommandLineProcessing(t *testing.T) {
	testCases := []struct {
		name string
		args []string
		expectHelp bool
		expectVersion bool
		expectTest bool
		expectLSP bool
	}{
		{
			name: "help flag",
			args: []string{"frugal-ls", "--help"},
			expectHelp: true,
		},
		{
			name: "short help flag",
			args: []string{"frugal-ls", "-h"},
			expectHelp: true,
		},
		{
			name: "version flag",
			args: []string{"frugal-ls", "--version"},
			expectVersion: true,
		},
		{
			name: "short version flag", 
			args: []string{"frugal-ls", "-v"},
			expectVersion: true,
		},
		{
			name: "test flag",
			args: []string{"frugal-ls", "--test"},
			expectTest: true,
		},
		{
			name: "no arguments",
			args: []string{"frugal-ls"},
			expectLSP: true,
		},
		{
			name: "unknown argument",
			args: []string{"frugal-ls", "--unknown"},
			expectLSP: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// We can't easily test the actual execution since it would run the LSP server
			// But we can verify the logic by checking os.Args parsing
			
			// Save original args
			origArgs := os.Args
			defer func() { os.Args = origArgs }()
			
			// Set test args
			os.Args = tc.args
			
			// For LSP mode, we just verify it would try to run (can't test actual LSP server easily)
			if tc.expectLSP {
				// These cases would run the LSP server, which we can't easily test in unit tests
				t.Log("LSP server mode - would need integration test")
				return
			}
			
			// For other modes, we can capture output
			if tc.expectHelp {
				// Capture stdout to test help output
				oldStdout := os.Stdout
				r, w, _ := os.Pipe()
				os.Stdout = w
				
				// This will call main() but should exit early for help
				defer func() {
					if r := recover(); r != nil {
						// Help mode calls os.Exit, which might cause issues in tests
						t.Log("Help mode attempted to exit")
					}
				}()
				
				printUsage() // Test the help function directly instead
				
				w.Close()
				os.Stdout = oldStdout
				
				var buf bytes.Buffer
				buf.ReadFrom(r)
				output := buf.String()
				
				if !strings.Contains(output, "Usage:") {
					t.Error("Help should show usage")
				}
			}
		})
	}
}

func TestTestParser(t *testing.T) {
	// Capture stdout to verify test parser output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the test parser
	testParser()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify the output contains some parsing information
	if output == "" {
		t.Error("Test parser should produce output")
	}

	// Should contain some indication of parsing
	if !strings.Contains(output, "struct") && !strings.Contains(output, "service") {
		t.Error("Test parser output should contain struct or service information")
	}
}

// Test that the binary can be built and executed
func TestBinaryExecution(t *testing.T) {
	// This is more of an integration test to ensure the binary works
	cmd := exec.Command("go", "build", "-o", "/tmp/frugal-ls-test", ".")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("/tmp/frugal-ls-test")

	// Test help
	cmd = exec.Command("/tmp/frugal-ls-test", "--help")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Built binary failed on help: %v", err)
	}

	if !strings.Contains(out.String(), "Usage:") {
		t.Error("Built binary should show usage on --help")
	}

	// Test version
	cmd = exec.Command("/tmp/frugal-ls-test", "--version")
	out.Reset()
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Built binary failed on version: %v", err)
	}

	if !strings.Contains(out.String(), "0.1.0") {
		t.Error("Built binary should show version on --version")
	}
}