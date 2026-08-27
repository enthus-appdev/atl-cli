package cmd

import (
	"bytes"
	"errors"
	"fmt"
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
	contextName, err := parsedInvocationContext(root, []string{"jira", "--context", "sandbox", "issue", "list"})
	if err != nil {
		t.Fatalf("parse --context: %v", err)
	}
	var during string
	if err := runWithInvocationContext(contextName, func() error {
		during = os.Getenv("ATLASSIAN_CONTEXT")
		return nil
	}); err != nil {
		t.Fatalf("execute with --context: %v", err)
	}
	if during != "sandbox" {
		t.Fatalf("ATLASSIAN_CONTEXT during command = %q, want sandbox", during)
	}
	if got := os.Getenv("ATLASSIAN_CONTEXT"); got != "from-environment" {
		t.Fatalf("ATLASSIAN_CONTEXT after command = %q, want restored value", got)
	}
}

func TestContextFlagRestoresOverrideAfterError(t *testing.T) {
	t.Setenv("ATLASSIAN_CONTEXT", "from-environment")
	wantErr := fmt.Errorf("command failed")
	if err := runWithInvocationContext("sandbox", func() error {
		if got := os.Getenv("ATLASSIAN_CONTEXT"); got != "sandbox" {
			t.Fatalf("ATLASSIAN_CONTEXT during command = %q, want sandbox", got)
		}
		return wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("runWithInvocationContext() error = %v, want %v", err, wantErr)
	}
	if got := os.Getenv("ATLASSIAN_CONTEXT"); got != "from-environment" {
		t.Fatalf("ATLASSIAN_CONTEXT after error = %q, want restored value", got)
	}
}

func TestParsedInvocationContext(t *testing.T) {
	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"--context", "prod", "jira", "issue", "list"}, want: "prod"},
		{args: []string{"jira", "--context=sandbox", "assets", "count"}, want: "sandbox"},
		{args: []string{"jira", "issue", "comment", "add", "NX-1", "--body", "--context=positional"}, want: ""},
		{args: []string{"jira", "issue", "create", "--", "--context=positional"}, want: ""},
		{args: []string{"jira", "issue", "list"}, want: ""},
	} {
		root := NewRootCmd(iostreams.Test(), "test")
		got, err := parsedInvocationContext(root, test.args)
		if err != nil {
			t.Errorf("parsedInvocationContext(%q) error: %v", test.args, err)
			continue
		}
		if got != test.want {
			t.Errorf("parsedInvocationContext(%q) = %q, want %q", test.args, got, test.want)
		}
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
