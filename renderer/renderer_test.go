package renderer

import (
	"strings"
	"testing"
)

func TestRender_RendersMarkdown(t *testing.T) {
	input := "# Hello\n\nThis is a test."
	result, err := Render(input, Options{NoColor: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(result, "Hello") {
		t.Error("expected 'Hello' in rendered output")
	}
	if !strings.Contains(result, "This is a test") {
		t.Error("expected 'This is a test' in rendered output")
	}
}

func TestRender_NoColorExcludesANSIEscapeSequences(t *testing.T) {
	input := "# Title\n\n**bold text**"
	result, err := Render(input, Options{NoColor: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// ANSI escape sequences start with \x1b[
	if strings.Contains(result, "\x1b[") {
		t.Error("expected no ANSI escape sequences in no-color mode")
	}
}

func TestRender_HandlesEmptyStringWithoutError(t *testing.T) {
	result, err := Render("", Options{NoColor: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Should return something (at least newline)
	_ = result
}
