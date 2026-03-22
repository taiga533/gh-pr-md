package renderer

import (
	"os"
	"regexp"
	"strings"

	"github.com/cli/go-gh/v2/pkg/markdown"
	"golang.org/x/term"
)

var ansiSGRRe = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// Options holds rendering options.
type Options struct {
	NoColor bool
}

// stripTrailingPadding removes Glamour's line-filling padding while
// preserving styled content like inline code with background colors.
// It scans each line left-to-right, tracks whether a background color
// is active, and records the last position that contains meaningful
// content. Trailing whitespace without background is removed.
func stripTrailingPadding(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = stripLinePadding(line)
	}
	return strings.Join(lines, "\n")
}

// stripLinePadding strips trailing padding from a single line.
func stripLinePadding(line string) string {
	bgActive := false
	lastKeep := 0

	for pos := 0; pos < len(line); {
		if line[pos] == 0x1b {
			loc := ansiSGRRe.FindStringSubmatchIndex(line[pos:])
			if loc != nil && loc[0] == 0 {
				params := line[pos+loc[2] : pos+loc[3]]
				end := pos + loc[1]

				switch {
				case params == "0" || params == "":
					if bgActive {
						lastKeep = end
					}
					bgActive = false
				case strings.Contains(params, "48;"):
					bgActive = true
					lastKeep = end
				default:
					if bgActive {
						lastKeep = end
					}
				}

				pos = end
				continue
			}
		}

		if bgActive || (line[pos] != ' ' && line[pos] != '\t') {
			lastKeep = pos + 1
		}
		pos++
	}

	result := line[:lastKeep]
	if strings.Contains(result, "\x1b[") && !strings.HasSuffix(result, "\x1b[0m") {
		result += "\x1b[0m"
	}
	return result
}

// splitFrontmatter separates YAML frontmatter from the rest of the markdown.
// Returns the frontmatter block (including delimiters) and the remaining body.
func splitFrontmatter(text string) (frontmatter, body string) {
	if !strings.HasPrefix(text, "---\n") {
		return "", text
	}
	end := strings.Index(text[4:], "\n---")
	if end == -1 {
		return "", text
	}
	// end is relative to text[4:]; closing delimiter "---" starts at 4+end+1
	closingStart := 4 + end + 1
	// Find the end of the "---" line
	lineEnd := strings.Index(text[closingStart:], "\n")
	var split int
	if lineEnd == -1 {
		split = len(text)
	} else {
		split = closingStart + lineEnd + 1
	}
	return text[:split], text[split:]
}

// Render renders a markdown string with optional ANSI color output.
// YAML frontmatter is preserved as-is without Glamour rendering.
func Render(text string, opts Options) (string, error) {
	frontmatter, body := splitFrontmatter(text)

	width := terminalWidth()
	theme := "dark"
	if opts.NoColor {
		theme = "notty"
	}
	rendered, err := markdown.Render(body,
		markdown.WithTheme(theme),
		markdown.WithWrap(width),
		markdown.WithoutIndentation(),
	)
	if err != nil {
		return "", err
	}
	cleaned := stripTrailingPadding(rendered)
	return frontmatter + cleaned, nil
}

// terminalWidth returns the current terminal width, defaulting to 120.
func terminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 120
	}
	return w
}
