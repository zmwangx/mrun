package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/zmwangx/mrun"
)

func main() {
	tmpdir, err := os.MkdirTemp("", "mrun-test-*")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpdir) }()
	commands := []*mrun.Command{
		// Empty command to test error handling.
		mrun.NewCommand(
			&exec.Cmd{},
			mrun.WithLabel("empty exec.Cmd"),
		),
		mrun.NewCommandWithShell(
			"bat --color=always --terminal-width=80 model.go | perl -pe 'select undef,undef,undef,0.1'", // Auto-scroll at 10 lines per second.
			mrun.WithLabel("model.go"),
		),
		mrun.NewCommand(exec.Command("ping", "-c", "5", "www.douyin.com")),
		// Test non-zero exit code.
		mrun.NewCommand(exec.Command("bash", "-c", "exit 1")),
		// Test nonexistent command.
		mrun.NewCommand(exec.Command("base", "-c", "exit 1")),
	}
	// Compile testprog.
	srcdir := "./cmd/testprog"
	_, err = os.Stat(srcdir)
	if err != nil {
		log.Fatalf("%s not found, make sure you are running this from the root of the repository", srcdir)
	}
	testprog := filepath.Join(tmpdir, "testprog")
	cmd := exec.Command("go", "build", "-C", srcdir, "-o", testprog, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		commands = append(commands, mrun.NewCommand(
			exec.Command("./testprog"),
			mrun.WithDir(tmpdir),
			mrun.WithLabel(fmt.Sprintf("test %d", i)),
		))
	}
	commands, ok, err := mrun.Run(
		commands,
		mrun.WithColumns(2),
		mrun.WithCommandLines(),
		mrun.WithFinalView(),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("all successful: %t\n", ok)
	for _, c := range commands {
		cerr := c.Err()
		if cerr == nil {
			fmt.Printf("%s: ok\n", c.CommandLine())
		} else {
			fmt.Printf("%s: %s\n", c.CommandLine(), cerr)
		}
	}
}
