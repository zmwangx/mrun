package mrun

import (
	"os"
	"os/exec"

	"al.essio.dev/pkg/shellescape"
)

type Command struct {
	cmd     *exec.Cmd
	cmdline string
	label   string
	done    bool
	// The error from running the command will be stored here.
	err error
}

type CommandOption func(*Command)

// NewCommand creates a new command to be run in its own pane. cmd can have its
// own Env and Dir. One can also use [WithEnv] and [WithDir] to set them for
// convenience.
//
// To add a label to the bottom of the pane, use [WithLabel].
//
// To set a custom command line (see [WithCommandLines]), use [WithCommandLine].
// Useful for hiding unnecessary details like a shell invocation.
//
// See also [NewCommandWithShell].
func NewCommand(cmd *exec.Cmd, opts ...CommandOption) *Command {
	c := &Command{cmd: cmd}
	for _, opt := range opts {
		opt(c)
	}
	if c.cmdline == "" {
		c.cmdline = shellescape.QuoteCommand(cmd.Args)
	}
	return c
}

// WithLabel adds a label that is shown at the bottom of the pane.
func WithLabel(label string) CommandOption {
	return func(c *Command) {
		c.label = label
	}
}

// WithCommandLine sets the command line to be shown at the bottom of the pane
// when [WithCommandLines] is used. By default it is auto-generated from the
// *exec.Cmd.
func WithCommandLine(cmdline string) CommandOption {
	return func(c *Command) {
		c.cmdline = cmdline
	}
}

// WithEnv sets the environment of the command.
//
// It's a convenience function equivalent to manually setting the Env field of
// cmd before calling NewCommand:
//
//	cmd.Env = append(os.Environ(), env...)
//
// Note: you need to manually set Env on the exec.Cmd if you don't want to
// inherit the current process's environment.
func WithEnv(env []string) CommandOption {
	return func(c *Command) {
		c.cmd.Env = append(os.Environ(), env...)
	}
}

// WithDir sets the working directory of the command.
//
// It's a convenience function equivalent to manually setting the Dir field of
// cmd before calling NewCommand:
//
//	cmd.Dir = dir
func WithDir(dir string) CommandOption {
	return func(c *Command) {
		c.cmd.Dir = dir
	}
}

// CommandLine returns the set or generated command line of the command.
func (c Command) CommandLine() string {
	return c.cmdline
}

// ProcessState returns the exit state of the command, if the command was
// successfully started and waited for.
func (c Command) ProcessState() *os.ProcessState {
	return c.cmd.ProcessState
}

// Err returns the error from running the command (including non-zero exit).
func (c Command) Err() error {
	return c.err
}
