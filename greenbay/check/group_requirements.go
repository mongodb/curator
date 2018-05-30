package check

import "github.com/pkg/errors"

// this is populated in init.go's init(), to avoid init() ordering
// effects. Only used during the init process, so we don't need locks
// for this.
var groupRequirementRegistry map[string]GroupRequirements

// GroupRequirements provides a way to express expected outcomes for
// checks that include multiple constituent checks. (e.g. a list of
// files that must exist or a list of commands to run.)
type GroupRequirements struct {
	Any  bool
	One  bool
	None bool
	All  bool
	Name string
}

// GetResults takes numbers of passed and failed tasks and reports if
// the check should succeed.
func (gr GroupRequirements) GetResults(passes, failures int) (bool, error) {
	if gr.All {
		if failures > 0 {
			return false, nil
		}
	} else if gr.One {
		if passes != 1 {
			return false, nil
		}
	} else if gr.Any {
		if passes == 0 {
			return false, nil
		}
	} else if gr.None {
		if passes > 0 {
			return false, nil
		}
	} else {
		return false, errors.Errorf("incorrectly configured group check for %s", gr.Name)
	}

	return true, nil
}

// Validate checks the GroupRequirements structure and ensures that
// the specified results are valid and that there are no contradictory
// or impossible expectations.
func (gr GroupRequirements) Validate() error {
	if gr.Name == "" {
		return errors.New("no name specified for group requirements specification")
	}

	opts := []bool{gr.All, gr.Any, gr.One, gr.None}
	active := 0

	for _, opt := range opts {
		if opt {
			active++
		}
	}

	if active != 1 {
		return errors.Errorf("specified incorrect number of options for a '%s' check: "+
			"[all=%t, one=%t, any=%t, none=%t]", gr.Name,
			gr.All, gr.One, gr.Any, gr.None)
	}

	return nil
}
