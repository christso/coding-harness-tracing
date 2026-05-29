// Package main is the entry point for the ax-trace CLI.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
)

var rootCmd = &cobra.Command{
	Use:   "ax-trace",
	Short: "Install and manage Arize coding-harness-tracing",
	Long: `ax-trace is a portable CLI for installing and managing
Arize coding-harness-tracing across Claude Code, Codex, Copilot,
Cursor, Gemini, and Kiro.`,
	SilenceUsage:  true,
	SilenceErrors: true, // main owns all error output (avoids cobra double-printing)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		var ec *exitCodeError
		if errors.As(err, &ec) {
			os.Exit(ec.code) // specific status, no message
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
