package check

import (
	"os"
	"os/exec"
	"strings"

	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

func scriptCompilerInterfaceFactoryTable() map[string]compilerFactory {
	factory := func(path string) compilerFactory {
		return func() compiler {
			return &compileScript{
				bin: path,
			}
		}
	}

	return map[string]compilerFactory{
		"run-program-python-auto":      pythonCompilerAuto,
		"run-program-system-python":    factory("python"),
		"run-program-system-python2":   factory("python2"),
		"run-program-system-python3":   factory("python3"),
		"run-program-usr-bin-pypy":     factory("/usr/bin/pypy"),
		"run-program-usr-local-python": factory("/usr/local/bin/python"),
		"run-bash-script":              factory("/bin/bash"),
		"run-sh-script":                factory("/bin/sh"),
		"run-dash-script":              factory("/bin/dash"),
		"run-zsh-script":               factory("/bin/zsh"),
	}
}

type compileScript struct {
	bin string
}

func pythonCompilerAuto() compiler {
	c := compileScript{}

	paths := []string{
		"/opt/mongodbtoolchain/v2/bin/python",
		"/usr/local/bin/python",
		"/usr/bin/python",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			c.bin = path
			break
		}
	}

	if c.bin == "" {
		c.bin = "python"
	}

	return c
}

func (c compileScript) Validate() error {
	if c.bin == "" {
		return errors.New("no script interpreter")
	}

	return nil
}

func (c compileScript) Compile(testBody string, _ ...string) error {
	_, sourceName, err := writeTestBody(testBody, "py")
	if err != nil {
		return errors.Wrap(err, "problem writing test")
	}

	defer os.Remove(sourceName)

	cmd := exec.Command(c.bin, sourceName)
	grip.Infof("running script script with command: %s", strings.Join(cmd.Args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "problem build/running test script %s: %s", sourceName,
			string(output))
	}

	return nil
}

func (c compileScript) CompileAndRun(testBody string, _ ...string) (string, error) {
	_, sourceName, err := writeTestBody(testBody, "py")
	if err != nil {
		return "", errors.Wrap(err, "problem writing test")
	}

	defer os.Remove(sourceName)

	cmd := exec.Command(c.bin, sourceName)
	grip.Infof("running script script with command: %s", strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil {
		return output, errors.Wrapf(err, "problem running test script %s", sourceName)
	}

	return strings.Trim(output, "\r\t\n "), nil
}
