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
	if strings.Contains(result, "\x1b[") {
		t.Error("expected no ANSI escape sequences in no-color mode")
	}
}

func TestRender_HandlesEmptyStringWithoutError(t *testing.T) {
	result, err := Render("", Options{NoColor: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result even for empty input")
	}
}

func TestStripLinePadding_StripsTrailingPaddingButPreservesBackgroundColoredSpaces(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain trailing whitespace is stripped",
			input: "some text   ",
			want:  "some text",
		},
		{
			name:  "foreground-only ANSI padding is stripped",
			input: "\x1b[38;5;252mtext\x1b[0m\x1b[38;5;252m \x1b[0m\x1b[38;5;252m \x1b[0m",
			want:  "\x1b[38;5;252mtext\x1b[0m",
		},
		{
			name:  "background-colored space is preserved",
			input: "\x1b[38;5;203;48;5;236m code \x1b[0m\x1b[38;5;252m \x1b[0m",
			want:  "\x1b[38;5;203;48;5;236m code \x1b[0m",
		},
		{
			name:  "all-padding line becomes empty",
			input: "\x1b[0m\x1b[38;5;252m \x1b[0m\x1b[38;5;252m \x1b[0m",
			want:  "",
		},
		{
			name:  "unclosed style gets reset appended",
			input: "\x1b[38;5;228;1mHeading",
			want:  "\x1b[38;5;228;1mHeading\x1b[0m",
		},
		{
			name:  "empty line stays empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripLinePadding(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitFrontmatter_SeparatesFrontmatterFromBody(t *testing.T) {
	input := "---\ntitle: \"hello\"\n---\n# Body\n"
	fm, body := splitFrontmatter(input)

	if fm != "---\ntitle: \"hello\"\n---\n" {
		t.Errorf("unexpected frontmatter: %q", fm)
	}
	if body != "# Body\n" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestSplitFrontmatter_ReturnsEmptyFrontmatterWhenNone(t *testing.T) {
	input := "# No frontmatter\n"
	fm, body := splitFrontmatter(input)

	if fm != "" {
		t.Errorf("expected empty frontmatter, got %q", fm)
	}
	if body != input {
		t.Errorf("expected body to equal input, got %q", body)
	}
}

func TestRender_PreservesFrontmatterAsIs(t *testing.T) {
	input := "---\nnumber: 42\ntitle: \"Test\"\n---\n**bold text**\n"
	result, err := Render(input, Options{NoColor: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.HasPrefix(result, "---\nnumber: 42\ntitle: \"Test\"\n---\n") {
		t.Errorf("expected frontmatter preserved at start, got:\n%s", result)
	}
}

func TestRender_ANSIStyleIsResetAtEndOfEachLine(t *testing.T) {
	input := "# Heading\n\nSome text with `inline code` here.\n\nText ending `code`\n"
	result, err := Render(input, Options{NoColor: false})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i, line := range strings.Split(result, "\n") {
		if !strings.Contains(line, "\x1b[") {
			continue
		}
		if !strings.HasSuffix(line, "\x1b[0m") {
			t.Errorf("line %d has unclosed ANSI style: %q", i, line)
		}
	}
}

func TestRender_PreservesInlineCodeBackgroundPaddingAtEndOfLine(t *testing.T) {
	input := "Text ending `code`\n"
	result, err := Render(input, Options{NoColor: false})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Inline code should have background color (48;5;...) in the output
	if !strings.Contains(result, "48;5;") {
		t.Error("expected inline code to have background color")
	}
	// The inline code should have padding space with background before the reset
	if !strings.Contains(result, "code \x1b[0m") {
		t.Errorf("expected inline code trailing space to be preserved, got:\n%q", result)
	}
}

func TestRender_ColorModeReturnsNonEmptyOutput(t *testing.T) {
	input := "# Title\n\n**bold text**"
	result, err := Render(input, Options{NoColor: false})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result in color mode")
	}
	if !strings.Contains(result, "Title") {
		t.Error("expected 'Title' in color rendered output")
	}
}
