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
	var err error
	err = ensureExistence("model.go")
	if err != nil {
		log.Fatal(err)
	}
	err = ensureExistence("exec.go")
	if err != nil {
		log.Fatal(err)
	}
	err = ensureExistence("cmd/testprog")
	if err != nil {
		log.Fatal(err)
	}

	tmpdir, err := os.MkdirTemp("", "mrun-demo-*")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpdir) }()

	commands := []*mrun.Command{
		mrun.NewCommandWithShell(
			"bat --color=always --terminal-width=80 model.go | perl -pe 'select undef,undef,undef,0.1'", // Auto-scroll at 10 lines per second.
			// Use mrun.WithLabel() to add a label to the bottom of the command's pane.
			mrun.WithLabel("model.go"),
		),
		mrun.NewCommandWithShell(
			"bat --color=always --terminal-width=80 exec.go | perl -pe 'select undef,undef,undef,0.1'", // Auto-scroll at 10 lines per second.
			mrun.WithLabel("exec.go"),
		),
		mrun.NewCommand(
			exec.Command("git", "clone", "https://github.com/golang/go"),
			// Set environment variables.
			mrun.WithEnv([]string{"GIT_HTTP_LOW_SPEED_LIMIT=10000"}),
			// Change working directory of the command.
			mrun.WithDir(tmpdir),
		),
		mrun.NewCommand(
			exec.Command("yt-dlp", "https://www.bilibili.com/video/BV1JBcdeEEHU/"),
			mrun.WithDir(tmpdir),
		),
	}
	// Compile testprog.
	testprog := filepath.Join(tmpdir, "testprog")
	err = exec.Command("go", "build", "-C", "cmd/testprog", "-o", testprog, ".").Run()
	if err != nil {
		log.Fatal(err)
	}
	commands = append(commands, mrun.NewCommand(
		exec.Command("./testprog"),
		mrun.WithDir(tmpdir),
	))
	commands, ok, err := mrun.Run(
		commands,
		// Use mrun.WithColumns() to set the number of columns in the grid.
		mrun.WithColumns(2),
		// Use mrun.WithCommandLines() to print the command line before command
		// output in each pane.
		mrun.WithCommandLines(),
		// Use mrun.WithAutoQuit() to quit automatically after all commands are done.
		mrun.WithAutoQuit(),
		// Use mrun.WithFinalView() to leave a final, non-interactive view of
		// the grid on screen after quitting.
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

func ensureExistence(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s not found, make sure you are running this from the root of the repository", path)
	}
	return nil
}
