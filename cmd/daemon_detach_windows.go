//go:build windows

package cmd

import "os/exec"

// detachProcess no-op en Windows: el proceso hijo se independiza de la consola
// con los flags de creación por defecto de Go.
func detachProcess(cmd *exec.Cmd) {}
