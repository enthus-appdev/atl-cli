package cmd

import (
	"bytes"
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
		if c.PersistentPreRun == nil {
			t.Errorf("alias %q should have a deprecation hook", name)
		}
	}
}

func TestDeprecationWarning(t *testing.T) {
	var errOut bytes.Buffer
	ios := iostreams.Test()
	ios.ErrOut = &errOut

	root := NewRootCmd(ios, "test")
	alias := findChild(root, "issue")
	if alias == nil || alias.PersistentPreRun == nil {
		t.Fatal("issue alias or its deprecation hook missing")
	}

	alias.PersistentPreRun(alias, nil)

	if got := errOut.String(); !strings.Contains(got, "deprecated") || !strings.Contains(got, "jira issue") {
		t.Errorf("warning missing expected text, got: %q", got)
	}
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
