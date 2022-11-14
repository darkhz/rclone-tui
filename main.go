package main

import (
	"fmt"
	"os"

	"github.com/darkhz/rclone-tui/cmd"
	"github.com/darkhz/rclone-tui/ui"
)

func errMessage(err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
}

func main() {
	err := cmd.SetupConfig()
	if err != nil {
		errMessage(err)
		return
	}

	err = cmd.ParseFlags()
	if err != nil {
		errMessage(err)
		return
	}

	ui.SetupUI()
}
