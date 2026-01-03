package main

import (
	"os"

	"github.com/timkrase/deutsche-bahn-skill/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Runner{Version: version}))
}
