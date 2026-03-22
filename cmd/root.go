package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/taiga533/gh-pr-md/formatter"
	"github.com/taiga533/gh-pr-md/ghapi"
	"github.com/taiga533/gh-pr-md/renderer"
	"github.com/taiga533/gh-pr-md/resolver"
	"golang.org/x/term"
)

// Options holds the CLI flags for the pr-md command.
type Options struct {
	Repo    string
	NoDiff  bool
	NoColor bool
}

var opts Options

var rootCmd = &cobra.Command{
	Use:   "gh-pr-md [<number> | <url> | <branch>]",
	Short: "Display pull request information as formatted markdown",
	Long: `Display pull request information including review comments and
inline comments as formatted markdown in the terminal.

If no argument is provided, the PR is auto-detected from the current branch.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(args)
	},
	SilenceUsage: true,
}

func init() {
	rootCmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Select another repository using the [HOST/]OWNER/REPO format")
	rootCmd.Flags().BoolVar(&opts.NoDiff, "no-diff", false, "Exclude diff hunks from inline and review comments")
	rootCmd.Flags().BoolVar(&opts.NoColor, "no-color", false, "Disable ANSI color output")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func run(args []string) error {
	// Create GitHub API client
	client, err := ghapi.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Resolve PR target
	arg := ""
	if len(args) > 0 {
		arg = args[0]
	}
	result, err := resolver.Resolve(arg, opts.Repo, client)
	if err != nil {
		return err
	}

	// Fetch PR data
	pr, err := client.FetchPR(result.Owner, result.Repo, result.PRNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch PR #%d: %w", result.PRNumber, err)
	}

	// Format as markdown
	md := formatter.Format(pr, formatter.Options{
		NoDiff: opts.NoDiff,
	})

	// If stdout is not a TTY (piped or redirected), output plain markdown
	if !term.IsTerminal(int(os.Stdout.Fd())) && !opts.NoColor {
		fmt.Print(md)
		return nil
	}

	// Render to terminal
	rendered, err := renderer.Render(md, renderer.Options{
		NoColor: opts.NoColor,
	})
	if err != nil {
		return fmt.Errorf("failed to render markdown: %w", err)
	}

	fmt.Print(rendered)
	return nil
}
