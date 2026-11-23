//go:build !windows

package cmd

import (
	"os/exec"
	"syscall"
)

// detachProcess desacopla el daemon de la sesión y la terminal del padre para
// que no reciba SIGHUP al cerrar la shell.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
