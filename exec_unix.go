package mrun

import (
	"os/exec"
	"sync/atomic"
	"syscall"
	"time"
)

func (cmd *Command) gracefullyTerminate() {
	proc := cmd.cmd.Process
	if proc == nil {
		return
	}
	var done atomic.Bool
	go func() {
		if done.Load() {
			return
		}
		_ = proc.Signal(syscall.SIGINT)
		time.Sleep(3 * time.Second)
		if done.Load() {
			return
		}
		_ = proc.Signal(syscall.SIGTERM)
		time.Sleep(3 * time.Second)
		if done.Load() {
			return
		}
		_ = proc.Signal(syscall.SIGKILL)
	}()
	state, err := proc.Wait()
	if err != nil {
		cmd.err = err
	}
	// os.Process.Wait() doesn't return an error on non-zero exit status, so we
	// manually check and set here. Modeled on os/exec.Cmd.Wait():
	// https://cs.opensource.google/go/go/+/refs/tags/go1.23.5:src/os/exec/exec.go;l=898
	if !state.Success() {
		cmd.err = &exec.ExitError{ProcessState: state}
	}
	done.Store(true)
}
