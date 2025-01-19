package mrun

import "os/exec"

func (cmd *Command) gracefullyTerminate() {
	// os.Process.Signal() doesn't support SIGINT on Windows, but tt seems
	// graceful termination is possible on Windows:
	// - https://github.com/golang/go/issues/46345#issuecomment-847094650
	// - https://github.com/mattn/goreman/blob/e9150e84f13c37dff0a79b8faed5b86522f3eb8e/proc_windows.go#L16-L51
	// I can't be bothered for now.
	proc := cmd.cmd.Process
	if proc == nil {
		return
	}
	_ = proc.Kill()
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
}
