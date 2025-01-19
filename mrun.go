package mrun

// An example program demonstrating the pager component from the Bubbles
// component library.

import (
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone"
)

type runOpts struct {
	cols             int
	printCommandLine bool
	autoQuit         bool
	printFinalView   bool
}

type RunOption func(*runOpts)

// WithColumns sets the number of columns in the grid. The default is 1.
func WithColumns(cols int) RunOption {
	return func(o *runOpts) {
		o.cols = cols
	}
}

// WithCommandLines turns on printing the command line before command output in
// each pane.
func WithCommandLines() RunOption {
	return func(o *runOpts) {
		o.printCommandLine = true
	}
}

// WithAutoQuit turns on auto quitting after all commands are done. By default a
// dialog pops up asking if they want to quit.
func WithAutoQuit() RunOption {
	return func(o *runOpts) {
		o.autoQuit = true
	}
}

// WithFinalView leaves a final, non-interactive view of the grid on screen
// after quitting. The default behavior is that of a typical TUI program:
// nothing is left in the scroll buffer.
func WithFinalView() RunOption {
	return func(o *runOpts) {
		o.printFinalView = true
	}
}

// Run runs the given commands simultaneously in a TUI grid.
//
// Return values are:
//   - Slice of commands, now with checkable Err() and ProcessState().
//   - allSuccessful, only true if all commands ran to completion and exited with
//     0 (if the user prematurely quit, this will be false even if the terminated
//     commands responded with exit status 0 on SIGINT/SIGTERM). If it's false,
//     use Err() or ProcessState() on each command to determine which ones failed
//     and why.
//   - err is for error from the mrun runner itself, not including errors from
//     commands.
//
// Various options can be used to customize the experience:
//   - [WithColumns] sets the number of columns in the grid, default to 1.
//   - [WithCommandLines] turns on printing the command line before command output in
//     each pane.
//   - [WithAutoQuit] turns on auto quitting after all commands are done without user interaction.
//   - [WithFinalView] leaves a final, non-interactive view of the grid on screen after quitting.
func Run(commands []*Command, opts ...RunOption) (c []*Command, allSuccessful bool, err error) {
	c = commands
	var o runOpts
	o.cols = 1
	for _, opt := range opts {
		opt(&o)
	}

	if len(commands) == 0 {
		err = errors.New("commands must not be empty")
		return
	}
	if o.cols <= 0 {
		err = errors.New("columns must be positive")
		return
	}

	zone.NewGlobal()

	m := newModel(o.cols, commands, o)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	mm, err := p.Run()
	if err != nil {
		err = fmt.Errorf("bubbletea error: %s", err)
		return
	}
	allSuccessful = m.executor.allSuccessful()
	m, ok := mm.(model)
	if !ok {
		err = fmt.Errorf("bubbletea error: unexpected model type from Program.Run: expected %T, got %T", m, mm)
		return
	}
	// Reset window title.
	fmt.Print(ansi.SetWindowTitle(""))
	if o.printFinalView {
		fmt.Println(m.finalView())
	}
	return
}
