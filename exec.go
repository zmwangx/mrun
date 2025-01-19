package mrun

import (
	"bufio"
	"bytes"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
)

type cmdOutputMsg struct {
	paneIdx int
	ch      <-chan tea.Msg
	line    []byte
}

type cmdExitMsg struct {
	paneIdx  int
	exited   bool
	exitCode int
	errored  bool
	err      error
}

type (
	allDoneMsg       struct{}
	allTerminatedMsg struct{}
)

type multiExecutor struct {
	sync.Mutex
	cmds         []*Command
	terminating  atomic.Bool
	wg           sync.WaitGroup
	successCount int
}

func newMultiExecutor() *multiExecutor {
	return &multiExecutor{}
}

func (ex *multiExecutor) runCommand(paneIdx, w, h int, cmd *Command) (chan<- winsize, tea.Cmd) {
	if ex.terminating.Load() {
		return nil, nil
	}

	ex.Lock()
	ex.cmds = append(ex.cmds, cmd)
	ex.Unlock()

	ch := make(chan tea.Msg, 100)
	winsizeCh := make(chan winsize)
	sendOutput := func(line []byte) {
		ch <- cmdOutputMsg{
			paneIdx: paneIdx,
			ch:      ch,
			line:    line,
		}
	}
	ex.wg.Add(1)
	go func() {
		defer ex.wg.Done()
		defer func() { cmd.done = true }()

		handleError := func(exitErr error) {
			ch <- cmdOutputMsg{
				paneIdx: paneIdx,
				ch:      ch,
				line:    []byte(_errorStyle.Render(exitErr.Error())),
			}
			ch <- cmdExitMsg{
				paneIdx: paneIdx,
				errored: true,
				err:     exitErr,
			}
		}

		ptmx, err := pty.StartWithSize(cmd.cmd, &pty.Winsize{
			Rows: uint16(h),
			Cols: uint16(w),
		})
		if err != nil {
			cmd.err = err
			handleError(err)
			return
		}

		// Handle window resize.
		go func() {
			for ws := range winsizeCh {
				_ = pty.Setsize(ptmx, &pty.Winsize{
					Rows: uint16(ws.h),
					Cols: uint16(ws.w),
				})
			}
		}()

		scanner := bufio.NewScanner(ptmx)
		// Split on \r\n|\r|\n, and return the line as well as the line ending (\r
		// or \n is preserved, \r\n is collapsed to \n). Adaptation of
		// bufio.ScanLines.
		scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			if atEOF && len(data) == 0 {
				return 0, nil, nil
			}
			lfpos := bytes.IndexByte(data, '\n')
			crpos := bytes.IndexByte(data, '\r')
			if crpos >= 0 {
				if lfpos < 0 || lfpos > crpos+1 {
					// We have a CR-terminated "line".
					return crpos + 1, data[0 : crpos+1], nil
				}
				if lfpos == crpos+1 {
					// We have a CRLF-terminated line.
					return lfpos + 1, append(data[0:crpos], '\n'), nil
				}
			}
			if lfpos >= 0 {
				// We have a LF-terminated line.
				return lfpos + 1, data[0 : lfpos+1], nil
			}
			// If we're at EOF, we have a final, non-terminated line. Return it.
			if atEOF {
				return len(data), data, nil
			}
			// Request more data.
			return 0, nil, nil
		})
		for scanner.Scan() {
			sendOutput(scanner.Bytes())
		}

		if ex.terminating.Load() {
			// If we're terminating, the process is already waited in
			// cmd.gracefullyTerminate(), so we don't double-wait here,
			// otherwise we may get a spurious "wait: no child processes" error.
			return
		}

		err = cmd.cmd.Wait()
		if err != nil {
			cmd.err = err
			// Check if the err is a regular non-zero exit code.
			if _, ok := err.(*exec.ExitError); !ok {
				handleError(err)
				return
			}
		} else {
			ex.Lock()
			ex.successCount++
			ex.Unlock()
		}
		ch <- cmdExitMsg{
			paneIdx:  paneIdx,
			exited:   true,
			exitCode: cmd.cmd.ProcessState.ExitCode(),
		}
	}()
	return winsizeCh, func() tea.Msg {
		return cmdOutputMsg{
			paneIdx: paneIdx,
			ch:      ch,
		}
	}
}

// nextOutput returns either the next cmdOutputMsg (for the next line of output)
// or the final cmdExitMsg. When handling a cmdOutputMsg, the caller should call
// nextOutput to schedule the next message.
func (ex *multiExecutor) nextOutput(msg cmdOutputMsg) tea.Msg {
	return <-msg.ch
}

// waitForAllDone blocks until all commands have exited, then returns an
// allDoneMsg. Must not be called before all commands have been added with
// runCommand.
func (ex *multiExecutor) waitForAllDone() tea.Msg {
	ex.wg.Wait()
	return allDoneMsg{}
}

// terminateAll tries to gracefully terminate all running commands, then returns
// an allTerminatedMsg. It always returns after 10s even if the commands are
// somehow stuck even after SIGKILL. Must not be called before all commands have
// been added with runCommand.
func (ex *multiExecutor) terminateAll() tea.Msg {
	ex.terminating.Store(true)
	ex.Lock()
	defer ex.Unlock()
	for _, cmd := range ex.cmds {
		if cmd.done {
			continue
		}
		go func() {
			cmd.gracefullyTerminate()
		}()
	}
	done := make(chan struct{})
	go func() {
		ex.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
	}
	return allTerminatedMsg{}
}

func (ex *multiExecutor) allSuccessful() bool {
	return ex.successCount == len(ex.cmds)
}
