package generator

import (
	"github.com/evergreen-ci/shrub"
	"github.com/mongodb/jasper/buildsystem/model"
	"github.com/pkg/errors"
)

// Make represents an evergreen config generator for Make-based projects.
type Make struct {
	model.Make
}

// NewMake returns a generator for Make.
func NewMake(m model.Make) *Make {
	return &Make{
		Make: m,
	}
}

func (m *Make) Generate() (*shrub.Configuration, error) {
	conf, err := shrub.BuildConfiguration(func(c *shrub.Configuration) {
		for _, mv := range m.Variants {
			variant := c.Variant(mv.Name)
			variant.DistroRunOn = mv.Distros

			var tasksForVariant []*shrub.Task
			for _, mvt := range mv.Tasks {
				tasks, err := m.GetTasksFromRef(mvt)
				if err != nil {
					panic(err)
				}
				newTasks, err := m.generateVariantTasksForRef(c, mv, tasks)
				if err != nil {
					panic(err)
				}
				tasksForVariant = append(tasksForVariant, newTasks...)
			}

			getProjectCmd := shrub.CmdGetProject{
				Directory: m.WorkingDirectory,
			}

			if len(tasksForVariant) >= minTasksForTaskGroup {
				tg := c.TaskGroup(getTaskGroupName(mv.Name)).SetMaxHosts(len(tasksForVariant) / 2)
				tg.SetupTask = shrub.CommandSequence{getProjectCmd.Resolve()}

				for _, task := range tasksForVariant {
					_ = tg.Task(task.Name)
				}
				_ = variant.AddTasks(tg.GroupName)
			} else {
				for _, task := range tasksForVariant {
					task.Commands = append([]*shrub.CommandDefinition{getProjectCmd.Resolve()}, task.Commands...)
					_ = variant.AddTasks(task.Name)
				}
			}
		}
	})

	if err != nil {
		return nil, errors.Wrap(err, "generating evergreen configuration")
	}
	return conf, nil
}

func (m *Make) generateVariantTasksForRef(c *shrub.Configuration, mv model.MakeVariant, mts []model.MakeTask) ([]*shrub.Task, error) {
	var tasks []*shrub.Task
	for _, mt := range mts {
		cmds, err := m.subprocessExecCmds(mv, mt)
		if err != nil {
			return nil, errors.Wrap(err, "generating commands to run")
		}
		tasks = append(tasks, c.Task(getTaskName(mv.Name, mt.Name)).Command(cmds...))
	}
	return tasks, nil
}

func (m *Make) subprocessExecCmds(mv model.MakeVariant, mt model.MakeTask) ([]shrub.Command, error) {
	env := model.MergeEnvironments(m.Environment, mv.Environment, mt.Environment)
	var cmds []shrub.Command
	for _, target := range mt.Targets {
		targetNames, err := m.GetTargetsFromRef(target)
		if err != nil {
			return nil, errors.Wrap(err, "resolving targets")
		}
		opts := target.Options.Merge(mt.Options, mv.Options)
		for _, targetName := range targetNames {
			cmds = append(cmds, &shrub.CmdExec{
				Binary:           "make",
				Args:             append(opts, targetName),
				Env:              env,
				WorkingDirectory: m.WorkingDirectory,
			})
		}
	}
	return cmds, nil
}
