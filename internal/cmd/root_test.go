package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

func TestJiraNamespace(t *testing.T) {
	root := NewRootCmd(iostreams.Test(), "test")

	// New canonical path resolves.
	if _, _, err := root.Find([]string{"jira", "issue", "view"}); err != nil {
		t.Fatalf("atl jira issue view did not resolve: %v", err)
	}
	for _, sub := range []string{"issue", "board", "sm"} {
		if _, _, err := root.Find([]string{"jira", sub}); err != nil {
			t.Errorf("atl jira %s did not resolve: %v", sub, err)
		}
	}
}

func TestDeprecatedTopLevelAliases(t *testing.T) {
	root := NewRootCmd(iostreams.Test(), "test")

	for _, name := range []string{"issue", "board", "sm"} {
		c := findChild(root, name)
		if c == nil {
			t.Errorf("top-level alias %q missing", name)
			continue
		}
		if !c.Hidden {
			t.Errorf("alias %q should be hidden", name)
		}
		if c.PreRun == nil {
			t.Errorf("alias %q should have a deprecation hook", name)
		}
	}
}

func TestDeprecationWarning(t *testing.T) {
	var errOut bytes.Buffer
	ios := iostreams.Test()
	ios.ErrOut = &errOut

	root := NewRootCmd(ios, "test")
	// A leaf under the alias, not the alias root — proves the warning reaches
	// the command actually executed (e.g. `atl issue view`).
	leaf := findChild(findChild(root, "issue"), "view")
	if leaf == nil || leaf.PreRun == nil {
		t.Fatal("issue view leaf or its deprecation hook missing")
	}

	leaf.PreRun(leaf, nil)

	if got := errOut.String(); !strings.Contains(got, "deprecated") || !strings.Contains(got, "jira issue") {
		t.Errorf("warning missing expected text, got: %q", got)
	}
}

func TestContextFlagSetsInvocationOverride(t *testing.T) {
	t.Setenv("ATLASSIAN_CONTEXT", "from-environment")
	root := NewRootCmd(iostreams.Test(), "test")
	var during string
	root.AddCommand(&cobra.Command{
		Use: "capture-context",
		Run: func(cmd *cobra.Command, args []string) {
			during = os.Getenv("ATLASSIAN_CONTEXT")
		},
	})
	root.SetArgs([]string{"capture-context", "--context", "sandbox"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute with --context: %v", err)
	}
	if during != "sandbox" {
		t.Fatalf("ATLASSIAN_CONTEXT during command = %q, want sandbox", during)
	}
	if got := os.Getenv("ATLASSIAN_CONTEXT"); got != "from-environment" {
		t.Fatalf("ATLASSIAN_CONTEXT after command = %q, want restored value", got)
	}
}

func TestSubcommandsDoNotOverrideRootPersistentHooks(t *testing.T) {
	root := NewRootCmd(iostreams.Test(), "test")
	var check func(*cobra.Command)
	check = func(parent *cobra.Command) {
		for _, child := range parent.Commands() {
			if child.PersistentPreRun != nil || child.PersistentPreRunE != nil ||
				child.PersistentPostRun != nil || child.PersistentPostRunE != nil {
				t.Errorf("%s overrides root invocation-context hooks", child.CommandPath())
			}
			check(child)
		}
	}
	check(root)
}

// findChild returns the direct subcommand with the given name, including hidden ones.
func findChild(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
