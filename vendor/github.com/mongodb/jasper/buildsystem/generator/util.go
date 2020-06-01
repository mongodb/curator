package generator

import (
	"strings"
)

const (
	// minTasksForTaskGroup is the minimum number of tasks that have to be in a
	// task group in order for it to be worthwhile to create a task group. Since
	// max hosts must can be at most half the number of tasks and we don't want
	// to use single-host task groups, we must have at least four tasks in the
	// group to make a multi-host task group.
	minTasksForTaskGroup = 4

	// taskGroupSuffix is used to name task groups.
	taskGroupSuffix = "_group"
)

// getTaskName returns an auto-generated task name.
func getTaskName(parts ...string) string {
	return strings.Join(parts, "-")
}

// getTaskGroupName returns an auto-generated task group name.
func getTaskGroupName(name string) string {
	return name + taskGroupSuffix
}
