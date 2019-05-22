// +build windows

package check

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/mongodb/grip"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows/registry"
)

func compilerInterfaceFactoryTable() map[string]compilerFactory {
	m := map[string]compilerFactory{
		"compile-visual-studio": newCompileVS,
	}

	// we have to add all UNIX compilers to the test registry so
	// that we can share configs between platforms with disjoint sets
	// of registered tests.
	for _, name := range []string{"compile-gcc-auto",
		"compile-gcc-system", "compile-toolchain-v2",
		"compile-toolchain-v1", "compile-toolchain-v0"} {

		m[name] = undefinedCompileCheckFactory(name)
	}

	return m
}

type compileVS struct {
	envVars  map[string][]string
	versions []string
	catcher  grip.Catcher
}

func newCompileVS() compiler {
	c := &compileVS{
		envVars: make(map[string][]string),
		catcher: grip.NewCatcher(),
	}

	var vcKeyName string
	if runtime.GOARCH == "amd64" {
		vcKeyName = "Software\\Wow6432Node\\Microsoft\\VisualStudio"
	} else {
		vcKeyName = "Software\\Microsoft\\VisualStudio"
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, vcKeyName,
		registry.ENUMERATE_SUB_KEYS|registry.READ)
	if err != nil {
		c.catcher.Add(errors.Wrap(err, "problem reading from registry"))
		return nil
	}
	defer k.Close()

	installed, err := k.ReadSubKeyNames(-1)
	if err != nil {
		c.catcher.Add(errors.Wrap(err, "problem reading subkeys"))
		return nil
	}

	for _, ver := range installed {
		pKeyname := fmt.Sprintf("%s\\%s\\Setup\\VC", vcKeyName, ver)
		subKey, err := registry.OpenKey(registry.LOCAL_MACHINE, pKeyname, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		defer subKey.Close()

		pDir, _, err := subKey.GetStringValue("ProductDir")
		if err != nil {
			continue
		}

		fullScript := fmt.Sprintf(`%svcvarsall.bat`, pDir)
		envData, err := exec.Command("cmd.exe", "/C", fullScript, runtime.GOARCH, "&", "set").CombinedOutput()
		if err != nil {
			continue
		}

		envLines := strings.Split(string(envData), "\r\n")
		// Some versions of visual studio print out non-environment variables
		// in their output - which causes an "Invalid parameter" error.
		// This loop ensures only env variables are captured.
		parsedEnvVars := make([]string, 0, len(envLines))
		for _, line := range envLines {
			if strings.Contains(line, "=") {
				parsedEnvVars = append(parsedEnvVars, line)
			}
		}
		c.envVars[ver] = parsedEnvVars
		c.versions = append(c.versions, ver)
	}
	sort.Strings(c.versions)

	return c
}

func (c *compileVS) findCl(envVars []string) (string, error) {
	var path string

	for _, vs := range c.envVars {
		for _, v := range vs {
			if (strings.HasPrefix(v, "PATH=") || strings.HasPrefix(v, "Path=")) &&
				strings.Contains(v, "Visual Studio") {
				path = v[5:]
				break
			}
		}
	}

	if path == "" {
		return "", errors.Errorf("Could not find the PATH in the VS environment variables")
	}

	for _, k := range strings.Split(path, ";") {
		testPath := fmt.Sprintf("%s\\cl.exe", k)
		if _, err := os.Stat(testPath); os.IsNotExist(err) {
			continue
		}

		return testPath, nil
	}

	return "", errors.Errorf("Could not find cl in PATH")
}

func (c *compileVS) compileOp(filename, version string, cFlags ...string) error {
	// If no version was specified, just use the latest version.
	if version == "" {
		if len(c.versions) == 0 {
			return errors.Errorf("Visual Studio is not installed on this system")
		}
		version = c.versions[len(c.versions)-1]
	}

	argv := []string{filename}
	argv = append(argv, cFlags...)

	envVars, ok := c.envVars[version]
	if !ok {
		return errors.Errorf("VisualStudio %s is not installed", version)
	}

	clPath, err := c.findCl(envVars)
	if err != nil {
		return err
	}

	cmd := exec.Command(clPath, argv...)
	cmd.Env = envVars
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Errorf("Compiler error (%v): %s", err, output)
	}

	return nil
}

func (c *compileVS) Validate() error {
	return c.catcher.Resolve()
}

func (c *compileVS) Compile(testBody string, cFlags ...string) error {
	outputName, sourceName, err := writeTestBody(testBody, "c")
	if err != nil {
		return fmt.Errorf("Error creating test body file: %v", err)
	}
	defer os.Remove(sourceName)

	argv := []string{fmt.Sprintf("/Fo%s", outputName)}
	argv = append(argv, cFlags...)
	argv = append(argv, "/c")

	err = c.compileOp(sourceName, "", argv...)
	if err != nil {
		return errors.Wrap(err, "problem compiling software")
	}
	defer grip.Warning(os.Remove(fmt.Sprintf("%s.obj", outputName)))

	return nil
}

func (c *compileVS) CompileAndRun(testBody string, cFlags ...string) (string, error) {
	outputName, sourceName, err := writeTestBody(testBody, "c")
	if err != nil {
		return "", errors.Wrap(err, "problem writing test to file")
	}

	defer os.Remove(sourceName)
	defer os.Remove(fmt.Sprintf("%s.obj", outputName))

	argv := []string{
		fmt.Sprintf("/Fo%s", outputName), // Set .obj output name
		fmt.Sprintf("/Fe%s", outputName), // Set .exe output name
	}
	argv = append(argv, cFlags...)
	err = c.compileOp(sourceName, "", argv...)
	if err != nil {
		return "", err
	}
	outputName = fmt.Sprintf("%s.exe", outputName)

	defer os.Remove(outputName)
	out, err := exec.Command(outputName).CombinedOutput()
	output := string(out)
	if err != nil {
		return output, errors.Wrap(err, "problem running test program")
	}

	output = strings.Trim(output, "\r\t\n ")
	return output, nil
}
