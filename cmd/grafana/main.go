package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matiasvillaverde/grafana-cli/internal/cli"
	"github.com/matiasvillaverde/grafana-cli/internal/config"
)

var exitFn = os.Exit

func run(args []string) int {
	path, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	app := cli.NewApp(config.NewFileStore(path))
	return app.Run(context.Background(), args)
}

func main() {
	exitFn(run(os.Args[1:]))
}
