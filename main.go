package main

import (
  "log"
  "os"

  "github.com/nabeo/cli-tool-example/add"
  "github.com/nabeo/cli-tool-example/list"
  "github.com/nabeo/cli-tool-example/delete"

  "github.com/urfave/cli/v2"
)

func main() {
  app := &cli.App{
    Flags: []cli.Flag{
      &cli.StringFlag{
        Name: "profile",
        Usage: "your aws profile",
      },
      &cli.StringFlag{
        Name: "conf",
        Usage: "path to config file",
      },
      &cli.BoolFlag{
        Name: "dryrun",
        Usage: "dry run",
      },
    },
    Commands: []*cli.Command{
      &add.Command,
      &delete.Command,
      &list.Command,
    },
  }

  err := app.Run(os.Args)

  if err != nil {
    log.Fatal(err)
  }
}
