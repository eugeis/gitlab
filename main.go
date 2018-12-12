package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/xanzy/go-gitlab"
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
			Name:  "cloneAll",
			Usage: "Clone all projects",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  flagGroup,
					Usage: "Gitlab group",
				}, cli.StringFlag{
					Name:  flagTarget,
					Usage: "Target script file",
				},
			},
			Action: func(c *cli.Context) (err error) {
				logrus.Infof("execute %v", c.Command.Name)

				client := gitlab.NewClient(nil, c.GlobalString(flagToken))
				if err = client.SetBaseURL(c.GlobalString(flagURL)); err != nil {
					return
				}

				var group *gitlab.Group
				if group, _, err = client.Groups.GetGroup(c.String(flagGroup)); err == nil {
					var file *os.File
					if file, err = os.Create(c.String(flagTarget)); err == nil {
						defer func() {
							err = file.Close()
							return
						}()
						w := bufio.NewWriter(file)
						if err = generateScriptTo(w, group, make(map[int]bool, 0), client); err == nil {
							err = w.Flush()
						}
					}
				}
				return
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Warn("exit because of error.")
	}
}

func generateScriptTo(writer *bufio.Writer, group *gitlab.Group, alreadyHandledGroupIds map[int]bool, client *gitlab.Client) (err error) {
	alreadyHandledGroupIds[group.ID] = true

	if _, err = writer.WriteString(fmt.Sprintf("\nmkdir \"%v\"\n", group.Path)); err != nil {
		return
	}
	if _, err = writer.WriteString(fmt.Sprintf("cd \"%v\"\n", group.Path)); err != nil {
		return
	}

	for _, project := range group.Projects {
		if _, err = writer.WriteString(fmt.Sprintf("git clone %v\n", project.HTTPURLToRepo)); err != nil {
			return
		}
		for _, sharedGroup := range project.SharedWithGroups {
			if !alreadyHandledGroupIds[sharedGroup.GroupID] {
				if group, _, err := client.Groups.GetGroup(sharedGroup.GroupID); err == nil {
					if err = generateScriptTo(writer, group, alreadyHandledGroupIds, client); err != nil {
						return
					}
				} else {
					logrus.Warn(err)
				}
			}
		}
	}
	if _, err = writer.WriteString("cd ..\n"); err != nil {
		return
	}
	return
}
