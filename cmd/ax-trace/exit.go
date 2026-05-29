package main

import "fmt"

// exitCodeError is returned by command handlers that need the process to exit
// with a specific non-zero status. main() inspects it via errors.As and calls
// os.Exit, so handlers stay testable and cobra's normal teardown still runs.
type exitCodeError struct{ code int }

func (e *exitCodeError) Error() string { return fmt.Sprintf("exited with code %d", e.code) }
