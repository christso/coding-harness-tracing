package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	axexec "github.com/Arize-ai/coding-harness-tracing/cmd/ax-trace/internal/exec"
	"github.com/Arize-ai/coding-harness-tracing/cmd/ax-trace/internal/manifest"
	"github.com/Arize-ai/coding-harness-tracing/cmd/ax-trace/internal/paths"
)

// uninstallSelections holds the per-harness --<harness> boolean flags, keyed
// by each harness's config.yaml alias (e.g. "claude-code"). Flags are
// registered dynamically from the manifest, so a new harness needs no changes
// here. When none are set, `ax-trace uninstall` does a full wipe; when one or
// more are set, only those harnesses are uninstalled and the shared runtime is
// left in place.
type uninstallSelections struct {
	flags map[string]*bool // config-key -> bound flag value
	order []string         // config-keys in registration order
}

// selected returns the config.yaml keys the user opted into via --<harness>
// flags, in registration order (deterministic).
func (s *uninstallSelections) selected() []string {
	var keys []string
	for _, key := range s.order {
		if *s.flags[key] {
			keys = append(keys, key)
		}
	}
	return keys
}

// harnessFlagSpec describes one --<harness> uninstall flag.
type harnessFlagSpec struct {
	key     string // config.yaml alias, e.g. "claude-code"
	display string // human name used in the flag description
}

// harnessFlagSpecs returns the harnesses to expose as uninstall flags, derived
// from the manifest (hyphenated config alias + display name). Falls back to
// defaultHarnessNames if the manifest can't be loaded, so --help still works.
func harnessFlagSpecs() []harnessFlagSpec {
	var specs []harnessFlagSpec
	if m, err := manifest.Load(); err == nil {
		for _, name := range m.HarnessNames() {
			key := strings.ReplaceAll(name, "_", "-")
			display := m.Harnesses[name].DisplayName
			if display == "" {
				display = key
			}
			specs = append(specs, harnessFlagSpec{key: key, display: display})
		}
		return specs
	}
	for _, name := range defaultHarnessNames {
		key := strings.ReplaceAll(name, "_", "-")
		specs = append(specs, harnessFlagSpec{key: key, display: key})
	}
	return specs
}

func init() {
	s := &uninstallSelections{flags: map[string]*bool{}}
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall selected harnesses, or all harnesses + the shared runtime",
		Long: `Uninstall coding-harness-tracing.

With no flags, uninstalls every installed harness and wipes the shared
Python runtime plus ax-trace's own state directory.

With one or more --<harness> flags, uninstalls only the selected harnesses
and leaves the shared runtime in place.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			keys := s.selected()
			if len(keys) == 0 {
				return runUninstallAll(cmd.Context())
			}
			return runUninstallSelected(cmd.Context(), keys)
		},
	}
	for _, h := range harnessFlagSpecs() {
		v := new(bool)
		s.flags[h.key] = v
		s.order = append(s.order, h.key)
		cmd.Flags().BoolVar(v, h.key, false, fmt.Sprintf("Uninstall %s tracing", h.display))
	}
	rootCmd.AddCommand(cmd)
}

// runUninstallSelected tears down each requested harness, continuing past
// per-harness failures so one broken install.py doesn't block the others.
// Leaves the shared runtime (venv, install dir, ax-trace state) in place.
//
// If the venv is missing, returns nil with a friendly note — there's
// nothing to tear down at the harness level.
func runUninstallSelected(ctx context.Context, keys []string) error {
	if !venvExists() {
		fmt.Fprintln(os.Stdout, "[ax-trace] venv not found — nothing to uninstall")
		return nil
	}

	installDir, err := paths.InstallDir()
	if err != nil {
		return fmt.Errorf("resolving install dir: %w", err)
	}

	var failed []string
	for _, key := range keys {
		installPy := filepath.Join(installDir, "tracing", harnessSubdir(key), "install.py")
		if _, statErr := os.Stat(installPy); statErr != nil {
			fmt.Fprintf(os.Stderr, "[ax-trace] %s install script not found at %s (skipping)\n", key, installPy)
			failed = append(failed, key)
			continue
		}
		fmt.Fprintf(os.Stdout, "[ax-trace] uninstalling %s tracing...\n", key)
		exitCode, dispatchErr := axexec.Dispatch(ctx, axexec.DispatchOptions{
			BinName: "python",
			Args:    []string{installPy, "uninstall"},
		})
		if dispatchErr != nil {
			fmt.Fprintf(os.Stderr, "[ax-trace] %s uninstall failed (continuing): %v\n", key, dispatchErr)
			failed = append(failed, key)
			continue
		}
		if exitCode != 0 {
			fmt.Fprintf(os.Stderr, "[ax-trace] %s uninstall exited with code %d (continuing)\n", key, exitCode)
			failed = append(failed, key)
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("one or more uninstalls failed: %v", failed)
	}
	return nil
}

// runUninstallAll mirrors install.sh's full-wipe path: tear down each
// installed harness, then run `python -m core.setup.wipe` to remove the
// shared runtime, then delete ax-trace's own state directory.
//
// If the venv is missing, skip directly to deleting the install + ax-trace
// directories.
func runUninstallAll(ctx context.Context) error {
	installDir, err := paths.InstallDir()
	if err != nil {
		return fmt.Errorf("resolving install dir: %w", err)
	}
	axTraceHome, err := paths.AxTraceHome()
	if err != nil {
		return fmt.Errorf("resolving ax-trace home: %w", err)
	}

	if !venvExists() {
		fmt.Fprintln(os.Stdout, "[ax-trace] venv not found — removing install directories")
		if err := os.RemoveAll(installDir); err != nil {
			return fmt.Errorf("removing %s: %w", installDir, err)
		}
		if err := os.RemoveAll(axTraceHome); err != nil {
			return fmt.Errorf("removing %s: %w", axTraceHome, err)
		}
		fmt.Fprintln(os.Stdout, "[ax-trace] uninstall complete.")
		return nil
	}

	harnesses, err := listInstalledHarnesses(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ax-trace] could not enumerate installed harnesses (continuing): %v\n", err)
	}
	for _, key := range harnesses {
		installPy := filepath.Join(installDir, "tracing", harnessSubdir(key), "install.py")
		if _, statErr := os.Stat(installPy); statErr != nil {
			fmt.Fprintf(os.Stderr, "[ax-trace] harness install script not found for %q (skipping): %v\n", key, statErr)
			continue
		}
		fmt.Fprintf(os.Stdout, "[ax-trace] uninstalling %s tracing...\n", key)
		exitCode, dispatchErr := axexec.Dispatch(ctx, axexec.DispatchOptions{
			BinName: "python",
			Args:    []string{installPy, "uninstall"},
		})
		if dispatchErr != nil {
			fmt.Fprintf(os.Stderr, "[ax-trace] %s uninstall failed (continuing): %v\n", key, dispatchErr)
			continue
		}
		if exitCode != 0 {
			fmt.Fprintf(os.Stderr, "[ax-trace] %s uninstall exited with code %d (continuing)\n", key, exitCode)
		}
	}

	fmt.Fprintln(os.Stdout, "[ax-trace] wiping shared runtime...")
	exitCode, err := axexec.Dispatch(ctx, axexec.DispatchOptions{
		BinName: "python",
		Args:    []string{"-m", "core.setup.wipe"},
	})
	if err != nil {
		return fmt.Errorf("wiping shared runtime: %w", err)
	}
	// Match install.sh's `set -e` behavior: if wipe fails (or the user
	// declines its confirmation prompt), abort before deleting ax-trace's own
	// state. This leaves breadcrumbs (bootstrap log, lock file) for debugging
	// a partial uninstall.
	if exitCode != 0 {
		return &exitCodeError{code: exitCode}
	}

	if err := os.RemoveAll(axTraceHome); err != nil {
		return fmt.Errorf("removing %s: %w", axTraceHome, err)
	}

	fmt.Fprintln(os.Stdout, "[ax-trace] uninstall complete.")
	return nil
}

// venvExists reports whether the venv's python interpreter is present on
// disk. Used to short-circuit uninstall when there's nothing to tear down.
func venvExists() bool {
	pyPath, err := paths.VenvPython()
	if err != nil {
		return false
	}
	_, statErr := os.Stat(pyPath)
	return statErr == nil
}
