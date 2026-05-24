package main

import (
	"os"

	"github.com/usuginus/go-rpcatlas/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if err != cli.ErrHelp {
			os.Exit(1)
		}
	}
}
