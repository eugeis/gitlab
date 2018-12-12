package main

import (
	"bufio"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
	"os"
)

type GitLabParams struct {
	Url       string
	GroupName string
	Target    string
	Token     string
}

type ScriptGenerator struct {
	client                 *gitlab.Client
	alreadyHandledGroupIds map[int]bool

	cloneWriter *bufio.Writer
	pullWriter  *bufio.Writer
}

func Generate(params *GitLabParams) (err error) {
	client := gitlab.NewClient(nil, params.Token)
	if err = client.SetBaseURL(params.Url); err != nil {
		return
	}

	var group *gitlab.Group
	if group, _, err = client.Groups.GetGroup(params.GroupName); err == nil {
		var pullFile *os.File
		var cloneFile *os.File

		if cloneFile, err = os.Create(
			fmt.Sprintf("%v/%v%v", params.Target, params.GroupName, "Clone.sh")); err != nil {
			return
		}

		if pullFile, err = os.Create(
			fmt.Sprintf("%v/%v%v", params.Target, params.GroupName, "Pull.sh")); err != nil {
			return
		}

		defer func() {
			err = cloneFile.Close()
			err = pullFile.Close()
			return
		}()

		generator := &ScriptGenerator{
			client:                 client,
			alreadyHandledGroupIds: make(map[int]bool, 0),
			cloneWriter:            bufio.NewWriter(cloneFile),
			pullWriter:             bufio.NewWriter(pullFile),
		}

		err = generator.generate(group)

		generator.flush()
	}
	return
}

func (o *ScriptGenerator) flush() (err error) {
	err = o.cloneWriter.Flush()
	err = o.pullWriter.Flush()
	return
}

func (o *ScriptGenerator) generate(group *gitlab.Group) (err error) {
	o.alreadyHandledGroupIds[group.ID] = true

	if err = o.writeMkdir(group); err != nil {
		return
	}

	if err = o.writeCd(group); err != nil {
		return
	}

	if group.Projects == nil {
		if group, _, err = o.client.Groups.GetGroup(group.ID); err != nil {
			return
		}
	}

	for _, project := range group.Projects {
		if err = o.writeCloneOrPull(project); err != nil {
			return
		}
		err = o.handleSharedGroups(project)
	}
	err = o.handleSubGroups(group.ID)

	if err = o.writeCdBack(); err != nil {
		return
	}
	return
}

func (o *ScriptGenerator) writeCdBack() (err error) {
	_, err = o.cloneWriter.WriteString("cd ..\n")
	_, err = o.pullWriter.WriteString("cd ..\n")
	return
}

func (o *ScriptGenerator) writeCloneOrPull(project *gitlab.Project) (err error) {
	_, err = o.cloneWriter.WriteString(fmt.Sprintf("git clone %v\n", project.SSHURLToRepo))
	_, err = o.pullWriter.WriteString(fmt.Sprintf("git -C %v pull\n", project.Path))
	return
}

func (o *ScriptGenerator) writeMkdir(group *gitlab.Group) (err error) {
	_, err = o.cloneWriter.WriteString(fmt.Sprintf("\nmkdir \"%v\"\n", group.Path))
	_, err = o.pullWriter.WriteString(fmt.Sprintf("\n"))
	return
}

func (o *ScriptGenerator) writeCd(group *gitlab.Group) (err error) {
	_, err = o.cloneWriter.WriteString(fmt.Sprintf("cd \"%v\"\n", group.Path))
	_, err = o.pullWriter.WriteString(fmt.Sprintf("cd \"%v\"\n", group.Path))
	return
}

func (o *ScriptGenerator) handleSubGroups(groupId int) (err error) {
	var subGroups []*gitlab.Group
	options := &gitlab.ListSubgroupsOptions{AllAvailable: new(bool)}
	subGroups, _, err = o.client.Groups.ListSubgroups(groupId, options)
	for _, subGroup := range subGroups {
		if !o.alreadyHandledGroupIds[subGroup.ID] {
			if err = o.generate(subGroup); err != nil {
				logrus.Warn(err)
			}
		}
	}
	return err
}

func (o *ScriptGenerator) handleSharedGroups(project *gitlab.Project) (err error) {
	var loadedGroup *gitlab.Group
	for _, sharedGroup := range project.SharedWithGroups {
		if !o.alreadyHandledGroupIds[sharedGroup.GroupID] {
			if loadedGroup, _, err = o.client.Groups.GetGroup(sharedGroup.GroupID); err == nil {
				if err = o.generate(loadedGroup); err != nil {
					logrus.Warn(err)
				}
			} else {
				logrus.Warn(err)
			}
		}
	}
	return
}
