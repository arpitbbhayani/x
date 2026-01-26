package main

import (
	"os"

	"github.com/REDFOX1899/ask-sh/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
