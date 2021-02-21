package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()

	app.Version = "0.0.1" //TODO use goreleaser for setting version
	app.Name = "torproxy"
	app.Usage = "Tor2Web reverse proxy for tdex clients to consume onion endpoints without installing a tor client"
	app.Commands = append(
		app.Commands,
		&start,
	)

	err := app.Run(os.Args)
	if err != nil {
		fatal(err)
	}
}

type invalidUsageError struct {
	ctx     *cli.Context
	command string
}

func (e *invalidUsageError) Error() string {
	return fmt.Sprintf("invalid usage of command %s", e.command)
}

func fatal(err error) {
	var e *invalidUsageError
	if errors.As(err, &e) {
		_ = cli.ShowCommandHelp(e.ctx, e.command)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "[torproxy] %v\n", err)
	}
	os.Exit(1)
}
