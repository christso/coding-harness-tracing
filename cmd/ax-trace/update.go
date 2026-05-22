package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Arize-ai/coding-harness-tracing/cmd/ax-trace/internal/bootstrap"
	axexec "github.com/Arize-ai/coding-harness-tracing/cmd/ax-trace/internal/exec"
	"github.com/Arize-ai/coding-harness-tracing/cmd/ax-trace/internal/paths"
)

var updateBranch string

func init() {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update coding-harness-tracing and re-register installed harnesses",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context())
		},
	}
	cmd.Flags().StringVar(&updateBranch, "branch", "main", "Git ref for the update")
	rootCmd.AddCommand(cmd)
}

func runUpdate(ctx context.Context) error {
	if _, err := bootstrap.Bootstrap(ctx, bootstrap.Options{Branch: updateBranch}); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}

	harnesses, err := listInstalledHarnesses(ctx)
	if err != nil {
		return fmt.Errorf("listing installed harnesses: %w", err)
	}
	if len(harnesses) == 0 {
		fmt.Fprintln(os.Stdout, "[ax-trace] no installed harnesses found to re-register")
		fmt.Fprintln(os.Stdout, "[ax-trace] update complete.")
		return nil
	}

	installDir, err := paths.InstallDir()
	if err != nil {
		return fmt.Errorf("resolving install dir: %w", err)
	}

	for _, key := range harnesses {
		installPy := filepath.Join(installDir, "tracing", harnessSubdir(key), "install.py")
		if _, statErr := os.Stat(installPy); statErr != nil {
			fmt.Fprintf(os.Stderr, "[ax-trace] harness install script not found for %q (skipping): %v\n", key, statErr)
			continue
		}
		fmt.Fprintf(os.Stdout, "[ax-trace] re-registering %s...\n", key)
		exitCode, err := axexec.Dispatch(ctx, axexec.DispatchOptions{
			BinName: "python",
			Args:    []string{installPy, "install"},
		})
		if err != nil {
			return fmt.Errorf("re-registering %s: %w", key, err)
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	}

	fmt.Fprintln(os.Stdout, "[ax-trace] update complete.")
	return nil
}

// listInstalledHarnesses calls core.setup.list_installed_harnesses via the venv
// python and returns the harness keys from config.yaml.
func listInstalledHarnesses(ctx context.Context) ([]string, error) {
	var buf bytes.Buffer
	exitCode, err := axexec.Dispatch(ctx, axexec.DispatchOptions{
		BinName: "python",
		Args: []string{
			"-c",
			"from core.setup import list_installed_harnesses as L\nprint(\"\\n\".join(L()))",
		},
		Stdout: &buf,
		Stderr: os.Stderr,
	})
	if err != nil {
		return nil, err
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("list_installed_harnesses exited with code %d", exitCode)
	}
	return parseHarnessList(buf.String()), nil
}

// parseHarnessList splits the newline-delimited output from
// list_installed_harnesses into a slice, skipping blank lines.
func parseHarnessList(out string) []string {
	var keys []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		keys = append(keys, line)
	}
	return keys
}

// harnessSubdir maps a harness key (as stored in config.yaml) to its
// tracing/<subdir> directory name. Mirrors core.setup.harness_dir: replace
// dashes with underscores (e.g. "claude-code" -> "claude_code").
func harnessSubdir(key string) string {
	return strings.ReplaceAll(key, "-", "_")
}
