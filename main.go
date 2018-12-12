package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

const flagDebug = "debug"
const flagToken = "token"
const flagURL = "url"
const flagGroup = "group"
const flagTarget = "target"

func main() {
	app := cli.NewApp()
	app.Usage = "Gitlab helper"
	app.Version = "1.0"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  flagToken,
			Usage: "Gitlab token",
		}, cli.BoolFlag{
			Name:  flagDebug,
			Usage: "Enable debug log level",
		}, cli.StringFlag{
			Name:  flagURL,
			Usage: "Base url",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "generateScripts",
			Usage: "GenerateScripts for clone, pull all projects of a group",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  flagGroup,
					Usage: "Gitlab group",
				}, cli.StringFlag{
					Name:  flagTarget,
					Usage: "Target dir",
				},
			},
			Action: func(c *cli.Context) (err error) {
				logrus.Infof("execute %v", c.Command.Name)

				err = Generate(&GitLabParams{
					Url:       c.GlobalString(flagURL),
					GroupName: c.String(flagGroup),
					Target:    c.String(flagTarget),
					Token:     c.GlobalString(flagToken)})

				return
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Warn("exit because of error.")
	}
}
