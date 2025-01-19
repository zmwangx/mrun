package mrun

import "os/exec"

// NewCommandWithShell takes a command line to be run with sh. It is a shorthand
// for
//
//	NewCommand(exec.Command("sh", "-c", cmdline), append([]CommandOption{WithCommandLine(cmdline)}, opts...)...)
func NewCommandWithShell(cmdline string, opts ...CommandOption) *Command {
	opts = append([]CommandOption{WithCommandLine(cmdline)}, opts...)
	return NewCommand(exec.Command("sh", "-c", cmdline), opts...)
}
