package mrun_test

import (
	"log"
	"os/exec"

	"github.com/zmwangx/mrun"
)

func Example() {
	commands, ok, err := mrun.Run(
		[]*mrun.Command{
			mrun.NewCommand(
				exec.Command("tail", "-f", "/var/log/service1.log"),
				mrun.WithLabel("service1.log"),
			),
			mrun.NewCommand(
				exec.Command("tail", "-f", "/var/log/service2.log"),
				mrun.WithLabel("service2.log"),
			),
			mrun.NewCommandWithShell(
				"tail -f /var/log/service3.log | grep --line-buffered ERROR",
				mrun.WithLabel("service3.log"),
			),
		},
		mrun.WithColumns(2),
		mrun.WithCommandLines(),
		// mrun.WithAutoQuit(),
		// mrun.WithFinalView(),
	)
	if err != nil {
		log.Fatal(err)
	}
	if !ok {
		log.Print("some commands failed")
	}
	for _, cmd := range commands {
		err = cmd.Err()
		if err != nil {
			log.Printf("%s: %s", cmd.CommandLine(), err)
		}
	}
}
