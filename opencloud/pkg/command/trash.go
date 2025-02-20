package command

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/trash"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"
	"github.com/urfave/cli/v2"
)

func TrashCommand(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "trash",
		Usage: "OpenCloud trash functionality",
		Subcommands: []*cli.Command{
			TrashPurgeEmptyDirsCommand(cfg),
		},
		Before: func(c *cli.Context) error {
			return configlog.ReturnError(parser.ParseConfig(cfg, true))
		},
		Action: func(_ *cli.Context) error {
			fmt.Println("Read the docs")
			return nil
		},
	}
}

func TrashPurgeEmptyDirsCommand(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "purge-empty-dirs",
		Usage: "purge empty directories",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "basepath",
				Aliases:  []string{"p"},
				Usage:    "the basepath of the decomposedfs (e.g. /var/tmp/opencloud/storage/users)",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "do not delete anything, just print what would be deleted",
				Value: true,
			},
		},
		Action: func(c *cli.Context) error {
			basePath := c.String("basepath")
			if basePath == "" {
				fmt.Println("basepath is required")
				return cli.ShowCommandHelp(c, "trash")
			}

			if err := trash.PurgeTrashEmptyPaths(basePath, c.Bool("dry-run")); err != nil {
				fmt.Println(err)
				return err
			}

			return nil
		},
	}
}

func init() {
	register.AddCommand(TrashCommand)
}
