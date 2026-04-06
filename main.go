package main

import (
	"fmt"
	"os"

	"github.com/topdata-software-gmbh/ip-sentry/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
