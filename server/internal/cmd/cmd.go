package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/urfave/cli/v3"
)

// NewAppCmd creates a new CLI application with commands for Power Monitor
func NewAppCmd() *cli.Command {
	serve := false

	cmd := &cli.Command{
		Name:  "power-monitor",
		Usage: "Power Monitor IoT data collection and visualization system",
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Start the Power Monitor server",
				Action: func(ctx context.Context, command *cli.Command) error {
					serve = true
					return nil
				},
			},
			// User management commands
			{
				Name:  "user",
				Usage: "User management commands",
				Commands: []*cli.Command{
					CreateUserCommand,
					ListUsersCommand,
					UpdateUserCommand,
					ResetPasswordCommand,
					DeleteUserCommand,
				},
			},
			// Collector management commands
			{
				Name:  "collector",
				Usage: "Collector management commands",
				Commands: []*cli.Command{
					AddCollectorCommand,
					ListCollectorsCommand,
					UpdateCollectorCommand,
					ConfigCollectorCommand,
					StatusCollectorCommand,
					DeleteCollectorCommand,
				},
			},
			// Registration code management commands
			{
				Name:  "regcode",
				Usage: "Registration code management commands",
				Commands: []*cli.Command{
					GenerateRegCodeCommand,
					ListRegCodesCommand,
					RevokeRegCodeCommand,
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Value: "app.ini",
				Usage: "configuration file path",
			},
		},
		DefaultCommand: "serve",
		Version:        "1.0.0",
	}

	// Set the version printer
	cli.VersionPrinter = VersionPrinter

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	} else if !serve {
		os.Exit(0)
	}
	return cmd
}

// VersionPrinter prints version information
func VersionPrinter(cmd *cli.Command) {
	fmt.Printf("%s %s (Go %s %s/%s)\n",
		cmd.Root().Name, cmd.Root().Version,
		runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Println(cmd.Root().Usage)
}
