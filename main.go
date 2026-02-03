package main

import (
	"os"

	"github.com/cychiuae/shhh/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
