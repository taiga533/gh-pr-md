package renderer

import (
	"os"
	"regexp"

	"github.com/cli/go-gh/v2/pkg/markdown"
	"golang.org/x/term"
)

// Matches trailing whitespace at end of lines, including whitespace
// wrapped in ANSI escape sequences (e.g. "\x1b[38;5;252m \x1b[0m").
var trailingSpacesRe = regexp.MustCompile(`(?m)(\x1b\[[0-9;]*m| |\t)+$`)

// Options holds rendering options.
type Options struct {
	NoColor bool
}

// Render renders a markdown string with optional ANSI color output.
func Render(text string, opts Options) (string, error) {
	width := terminalWidth()
	theme := "dark"
	if opts.NoColor {
		theme = "none"
	}
	rendered, err := markdown.Render(text,
		markdown.WithTheme(theme),
		markdown.WithWrap(width),
		markdown.WithoutIndentation(),
	)
	if err != nil {
		return "", err
	}
	return trailingSpacesRe.ReplaceAllString(rendered, ""), nil
}

// terminalWidth returns the current terminal width, defaulting to 120.
func terminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 120
	}
	return w
}
