package check

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

func goCompilerIterfaceFactoryTable() map[string]compilerFactory {
	systemPath := os.Getenv("PATH")

	factory := func(path string, envPath string) compilerFactory {
		return func() compiler {
			c := compileGolang{
				bin: path,
			}

			if envPath != "" {
				c.path = "PATH=" + strings.Join([]string{envPath, systemPath}, string(os.PathListSeparator))
			}

			return c
		}
	}

	return map[string]compilerFactory{
		"compile-go-auto":            goCompilerAuto,
		"compile-opt-go-default":     factory("/opt/go/bin/go", ""),
		"compile-toolchain-gccgo-v2": factory("/opt/mongodbtoolchain/v2/bin/go", "/opt/mongodbtoolchain/v2/bin"),
		"compile-usr-local-go":       factory("/usr/local/go", ""),
		"compile-user-local-go":      factory("/usr/bin/go", ""),
	}
}

func goCompilerAuto() compiler {
	paths := [][]string{
		{"/opt/go/bin/go", ""},
		{"/opt/mongodbtoolchain/v2/bin/go", "/opt/mongodbtoolchain/v2/bin"},
		{"/usr/bin/go", ""},
		{"/usr/local/go/bin/go", ""},
		{"/usr/local/bin/go", ""},
	}

	// pick up versioned golang toolchains (e.g. /opt/golang/go1.24),
	// preferring the newest version
	if golang, _ := filepath.Glob("/opt/golang/go*/bin/go"); len(golang) > 0 {
		sort.Strings(golang)
		newest := golang[len(golang)-1]
		root := filepath.Dir(filepath.Dir(newest))
		paths = append(paths, []string{newest, root})
	}
	c := compileGolang{}

	for _, path := range paths {
		if _, err := os.Stat(path[0]); !os.IsNotExist(err) {
			c.bin = path[0]
			if path[1] != "" {
				c.path = path[1]

			}
			break
		}
	}

	if c.bin == "" {
		c.bin = "go"
	}

	return c
}

type compileGolang struct {
	path string
	bin  string
}

func (c compileGolang) Validate() error {
	if c.bin == "" {
		return errors.New("no go binary specified")
	}

	// bare command names (e.g. "go") are resolved from PATH at compile
	// time by exec.Command, so validate them the same way
	if filepath.Base(c.bin) == c.bin {
		if _, err := exec.LookPath(c.bin); err != nil {
			return errors.Errorf("go binary '%s' not found in PATH", c.bin)
		}

		return nil
	}

	if _, err := os.Stat(c.bin); os.IsNotExist(err) {
		return errors.Errorf("go binary '%s' does not exist", c.bin)
	}

	return nil
}

func (c compileGolang) Compile(testBody string, _ ...string) error {
	_, source, err := writeTestBody(testBody, "go")
	if err != nil {
		return errors.Wrap(err, "writing test to temporary file")
	}
	defer func() { grip.Error(os.Remove(source)) }()

	cmd := exec.Command(c.bin, "build", source)
	if c.path != "" {
		cmd.Env = []string{c.path}
	}

	grip.Infof("running build command: %s", cmd.Args)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "compiling Go test: %s", string(out))
	}

	return nil
}

func (c compileGolang) CompileAndRun(testBody string, _ ...string) (string, error) {
	_, source, err := writeTestBody(testBody, "go")
	if err != nil {
		return "", errors.Wrap(err, "writing test to temporary file")
	}
	defer func() { grip.Error(os.Remove(source)) }()

	cmd := exec.Command(c.bin, "run", source)
	grip.Infof("running script: %s", cmd.Args)

	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil {
		return output, errors.Wrapf(err, "running Go program: %s", output)
	}

	output = strings.Trim(output, "\r\t\n ")
	return output, nil
}
