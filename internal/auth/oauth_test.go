package auth

import "testing"

func TestDefaultScopesIncludeAssetsReads(t *testing.T) {
	want := map[string]bool{
		"read:cmdb-object:jira": false,
		"read:cmdb-schema:jira": false,
	}
	for _, scope := range DefaultScopes() {
		if _, ok := want[scope]; ok {
			want[scope] = true
		}
	}
	for scope, found := range want {
		if !found {
			t.Errorf("DefaultScopes missing %s", scope)
		}
	}
}
