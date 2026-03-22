package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	testOpts := Options{}
	cmd := &cobra.Command{
		Use:  "gh-pr-md [<number> | <url> | <branch>]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		SilenceUsage: true,
	}
	cmd.Flags().StringVarP(&testOpts.Repo, "repo", "R", "", "repository")
	cmd.Flags().BoolVar(&testOpts.NoDiff, "no-diff", false, "no diff")
	cmd.Flags().BoolVar(&testOpts.NoColor, "no-color", false, "no color")
	return cmd
}

func TestRootCmd_ExecutesWithNoArgs(t *testing.T) {
	cmd := newTestCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRootCmd_ExecutesWithOneArg(t *testing.T) {
	cmd := newTestCmd()
	cmd.SetArgs([]string{"123"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRootCmd_ReturnsErrorWithTwoOrMoreArgs(t *testing.T) {
	cmd := newTestCmd()
	cmd.SetArgs([]string{"123", "456"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for too many arguments, got nil")
	}
}

func TestRootCmd_ParsesFlagsCorrectly(t *testing.T) {
	testOpts := Options{}
	cmd := &cobra.Command{
		Use:  "gh-pr-md [<number> | <url> | <branch>]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		SilenceUsage: true,
	}
	cmd.Flags().StringVarP(&testOpts.Repo, "repo", "R", "", "repository")
	cmd.Flags().BoolVar(&testOpts.NoDiff, "no-diff", false, "no diff")
	cmd.Flags().BoolVar(&testOpts.NoColor, "no-color", false, "no color")

	cmd.SetArgs([]string{"--repo", "owner/repo", "--no-diff", "--no-color", "42"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if testOpts.Repo != "owner/repo" {
		t.Errorf("expected repo 'owner/repo', got '%s'", testOpts.Repo)
	}
	if !testOpts.NoDiff {
		t.Error("expected NoDiff to be true")
	}
	if !testOpts.NoColor {
		t.Error("expected NoColor to be true")
	}
}

func TestRootCmd_AcceptsShortFlagR(t *testing.T) {
	testOpts := Options{}
	cmd := &cobra.Command{
		Use:  "gh-pr-md",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.Flags().StringVarP(&testOpts.Repo, "repo", "R", "", "repository")

	cmd.SetArgs([]string{"-R", "host/owner/repo"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if testOpts.Repo != "host/owner/repo" {
		t.Errorf("expected repo 'host/owner/repo', got '%s'", testOpts.Repo)
	}
}
