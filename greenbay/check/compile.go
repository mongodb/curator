package check

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

type compiler interface {
	Validate() error
	Compile(string, ...string) error
	CompileAndRun(string, ...string) (string, error)
}

type compilerFactory func() compiler

func writeTestBody(testBody, ext string) (string, string, error) {
	testFile, err := ioutil.TempFile(os.TempDir(), "testBody_")
	if err != nil {
		return "", "", err
	}

	baseName := testFile.Name()
	sourceName := strings.Join([]string{baseName, ext}, ".")

	if runtime.GOOS == "windows" {
		testBody = strings.Replace(testBody, "\n", "\r\n", -1)
	}

	_, err = testFile.Write([]byte(testBody))
	if err != nil {
		return "", "", errors.Wrap(err, "writing test to file")
	}
	defer grip.Warning(testFile.Close())

	if err = os.Rename(baseName, sourceName); err != nil {
		return "", "", errors.Wrap(err, "renaming file")

	}

	return baseName, sourceName, nil
}

func registerCompileChecks() {
	compileCheckFactoryFactory := func(name string, c compiler, shouldRun bool) func() amboy.Job {
		return func() amboy.Job {
			return &compileCheck{
				Base:          NewBase(name, 0),
				shouldRunCode: shouldRun,
				compiler:      c,
			}
		}
	}

	registrar := func(table map[string]compilerFactory) {
		var jobName string
		for name, factory := range table {
			for _, shouldRun := range []bool{true, false} {
				if shouldRun {
					jobName = strings.Replace(name, "compile-", "compile-and-run-", 1)
				} else {
					jobName = name
				}

				registry.AddJobType(jobName,
					compileCheckFactoryFactory(jobName, factory(), shouldRun))
			}
		}
	}

	registrar(compilerInterfaceFactoryTable())
	registrar(goCompilerIterfaceFactoryTable())
}

type compileCheck struct {
	Source        string   `bson:"source" json:"source" yaml:"source"`
	Cflags        []string `bson:"cflags" json:"cflags" yaml:"cflags"`
	CflagsCommand string   `bson:"cflags_command" json:"cflags_command" yaml:"cflags_command"`
	*Base         `bson:"metadata" json:"metadata" yaml:"metadata"`
	shouldRunCode bool
	compiler      compiler
}

func (c *compileCheck) Run(_ context.Context) {
	c.startTask()
	defer c.MarkComplete()

	if err := c.compiler.Validate(); err != nil {
		c.setState(false)
		c.AddError(err)
		return
	}

	cflags := []string{}
	if c.CflagsCommand != "" {
		output, err := exec.Command("net-snmp-config", "--agent-libs").CombinedOutput()
		if err != nil {
			c.setState(false)
			c.AddError(err)
			return
		}

		cflags = append(cflags, strings.Split(strings.TrimSpace(string(output)), " ")...)
	}

	if len(c.Cflags) >= 1 {
		cflags = append(cflags, c.Cflags...)
	}

	if c.shouldRunCode {
		if output, err := c.compiler.CompileAndRun(c.Source, cflags...); err != nil {
			c.setState(false)
			c.AddError(err)
			c.setMessage(output)
		} else {
			c.setState(true)
		}
	} else {
		if err := c.compiler.Compile(c.Source, cflags...); err != nil {
			c.setState(false)
			c.AddError(err)
		} else {
			c.setState(true)
		}
	}
}
