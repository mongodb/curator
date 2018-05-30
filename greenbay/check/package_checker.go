package check

import (
	"fmt"
	"os/exec"
	"strings"
)

type packageChecker func(string) (bool, string)

// this is populated in init.go's init(), to avoid init() ordering
// effects. Only used during the init process, so we don't need locks
// for this.
var packageCheckerRegistry map[string]packageChecker

func packageCheckerFactory(args []string) packageChecker {
	return func(name string) (bool, string) {
		localArgs := append(args, name)

		out, err := exec.Command(localArgs[0], localArgs[1:]...).CombinedOutput()
		output := strings.Trim(string(out), "\r\t\n ")

		if err != nil {
			return false, fmt.Sprintf("%s package '%s' is not installed (%s) (%+v): %s",
				localArgs[0], name, err, output, strings.Join(localArgs, " "))
		}

		return true, output
	}
}
