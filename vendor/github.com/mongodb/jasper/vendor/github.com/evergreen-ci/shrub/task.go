package shrub

type Task struct {
	Name             string           `json:"name" yaml:"name"`
	PriorityOverride int              `json:"priority,omitempty" yaml:"priority_override,omitempty"`
	Dependencies     []TaskDependency `json:"depends_on,omitempty" yaml:"dependencies,omitempty"`
	Commands         CommandSequence  `json:"commands" yaml:"commands"`
}

type TaskDependency struct {
	Name    string `json:"name" yaml:"name"`
	Variant string `json:"variant" yaml:"variant"`
}

func (t *Task) Command(cmds ...Command) *Task {
	for _, c := range cmds {
		if err := c.Validate(); err != nil {
			panic(err)
		}

		t.Commands = append(t.Commands, c.Resolve())
	}

	return t
}

func (t *Task) AddCommand() *CommandDefinition {
	c := &CommandDefinition{}
	t.Commands = append(t.Commands, c)
	return c
}

func (t *Task) Dependency(dep ...TaskDependency) *Task {
	t.Dependencies = append(t.Dependencies, dep...)
	return t
}

func (t *Task) Function(fns ...string) *Task {
	for _, fn := range fns {
		t.Commands = append(t.Commands, &CommandDefinition{
			FunctionName: fn,
		})
	}

	return t
}

func (t *Task) FunctionWithVars(id string, vars map[string]string) *Task {
	t.Commands = append(t.Commands, &CommandDefinition{
		FunctionName: id,
		Vars:         vars,
	})

	return t
}

func (t *Task) Priority(pri int) *Task { t.PriorityOverride = pri; return t }

type TaskGroup struct {
	GroupName     string          `json:"name" yaml:"name"`
	MaxHosts      int             `json:"max_hosts,omitempty" yaml:"max_hosts,omitempty"`
	SetupGroup    CommandSequence `json:"setup_group,omitempty" yaml:"setup_group,omitempty"`
	SetupTask     CommandSequence `json:"setup_task,omitempty" yaml:"setup_task,omitempty"`
	Tasks         []string        `json:"tasks" yaml:"tasks"`
	TeardownTask  CommandSequence `json:"teardown_task,omitempty" yaml:"teardown_task,omitempty"`
	TeardownGroup CommandSequence `json:"teardown_group,omitempty" yaml:"teardown_group,omitempty"`
	Timeout       CommandSequence `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

func (g *TaskGroup) Name(id string) *TaskGroup      { g.GroupName = id; return g }
func (g *TaskGroup) SetMaxHosts(num int) *TaskGroup { g.MaxHosts = num; return g }
func (g *TaskGroup) Task(id ...string) *TaskGroup   { g.Tasks = append(g.Tasks, id...); return g }
