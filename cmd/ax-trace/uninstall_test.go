package main

import (
	"reflect"
	"testing"
)

// TestUninstallSelections_Selected verifies selected() returns only the
// set flags, as config.yaml keys, in registration order.
func TestUninstallSelections_Selected(t *testing.T) {
	cc, cx, kr := false, true, true
	s := &uninstallSelections{
		flags: map[string]*bool{"claude-code": &cc, "codex": &cx, "kiro": &kr},
		order: []string{"claude-code", "codex", "kiro"},
	}
	got := s.selected()
	want := []string{"codex", "kiro"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("selected() = %v, want %v", got, want)
	}
}

// TestHarnessFlagSpecs_DerivesHyphenatedKeys verifies flags are derived from
// the manifest using the hyphenated config alias (claude-code), not the
// underscored package key (claude_code), and carry the manifest display name.
func TestHarnessFlagSpecs_DerivesHyphenatedKeys(t *testing.T) {
	byKey := map[string]string{}
	for _, s := range harnessFlagSpecs() {
		byKey[s.key] = s.display
	}
	if d, ok := byKey["claude-code"]; !ok || d != "Claude Code" {
		t.Errorf("claude-code spec = (%q, ok=%v), want display %q", d, ok, "Claude Code")
	}
	if _, ok := byKey["claude_code"]; ok {
		t.Error("underscored manifest key claude_code must not be exposed as a flag")
	}
}
