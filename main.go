package main

import (
	"os"

	"github.com/nitrocode/breakup/internal/cli"
)

func main() {
	os.Exit(cli.Run())
}
