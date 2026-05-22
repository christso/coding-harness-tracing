package main

import (
	"strings"
	"testing"
)

func TestUpdateAndUninstallRegistered(t *testing.T) {
	got := map[string]bool{}
	for _, cmd := range rootCmd.Commands() {
		got[cmd.Use] = true
	}
	if !got["update"] {
		t.Error("update command not registered")
	}
	found := false
	for use := range got {
		if strings.HasPrefix(use, "uninstall") {
			found = true
			break
		}
	}
	if !found {
		t.Error("uninstall command not registered")
	}
}
