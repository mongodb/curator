package bond

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// BuildInfo captures information about a specific, single, build, and
// includes information about the build variant (i.e. edition, target
// platform, and architecture) as well as the version.
type BuildInfo struct {
	Version string       `json:"version"`
	Options BuildOptions `json:"options"`
}

func (i *BuildInfo) String() string {
	out, err := json.MarshalIndent(i, "", "   ")
	if err != nil {
		return fmt.Sprintf("{ '%s': 'error' }", i.Version)
	}
	return string(out)
}

// GetInfoFromFileName given a path, parses information about the
// build from that string and returns a BuildInfo object.
func GetInfoFromFileName(fileName string) (BuildInfo, error) {
	info := BuildInfo{Options: BuildOptions{}}
	fileName = filepath.Base(fileName)

	if strings.Contains(fileName, "debugsymbols") {
		info.Options.Debug = true
	}

	for _, arch := range []MongoDBArch{AMD64, X86, POWER, ZSeries} {
		if strings.Contains(fileName, string(arch)) {
			info.Options.Arch = arch
			break
		}
	}

	if info.Options.Arch == "" {
		return BuildInfo{}, errors.Errorf("path '%s' does not specify an arch", fileName)
	}

	version, err := getVersion(fileName)
	if err != nil {
		return BuildInfo{}, errors.Wrap(err, "problem resolving version")
	}
	info.Version = version

	edition, err := getEdition(fileName)
	if err != nil {
		return BuildInfo{}, errors.Wrap(err, "problem resolving edition")
	}
	info.Options.Edition = edition

	target, err := getTarget(fileName, version)
	if err != nil {
		return BuildInfo{}, errors.Wrap(err, "problem resolving target")
	}
	info.Options.Target = target

	return info, nil
}

func getVersion(fn string) (string, error) {
	parts := strings.Split(fn, "-")
	if len(parts) <= 3 {
		return "", errors.Errorf("path %s does not contain enough elements to include a version", fn)
	}

	if strings.Contains(fn, "latest") {
		if strings.HasPrefix(parts[len(parts)-2], "v") {
			return strings.Join(parts[len(parts)-2:], "-"), nil
		}
		return "latest", nil
	} else if strings.Contains(fn, "rc") {
		return strings.Join(parts[len(parts)-1:], "-"), nil
	}

	return parts[len(parts)-1], nil
}

func getEdition(fn string) (MongoDBEdition, error) {
	if strings.Contains(fn, "enterprise") {
		return Enterprise, nil
	}

	for _, distro := range []string{"rhel", "suse", "osx-ssl", "debian", "ubuntu", "amazon"} {
		if strings.Contains(fn, distro) {
			return CommunityTargeted, nil
		}
	}

	for _, platform := range []string{"osx", "win32", "sunos5", "linux", "sunos6"} {
		if strings.HasPrefix(fn, "mongodb-"+platform) {
			return Base, nil
		}
	}

	return "", errors.Errorf("path %s does not have a valid edition", fn)
}

func getTarget(fn, version string) (string, error) {
	// enterprise targets:
	if strings.Contains(fn, "enterprise") {
		for _, platform := range []string{"windows", "osx"} {
			if strings.Contains(fn, platform) {
				return platform, nil
			}
		}
		if strings.Contains(fn, "linux") {
			return strings.Split(fn, "-")[4], nil
		}
	}

	// all base and community targeted cases

	// linux distros
	if strings.Contains(fn, "linux") {
		return "linux", nil
	}

	// all windows windows
	if strings.Contains(fn, "2008plus-ssl") {
		return "windows_x86_64-2008plus-ssl", nil
	}
	if strings.Contains(fn, "2008plus") {
		return "windows_x86_64-2008plus", nil
	}
	if strings.Contains(fn, "win32-i386") {
		return "windows_i686", nil
	}
	if strings.Contains(fn, "win32-x86_64") {
		return "windows_x86_64", nil
	}

	// OSX variants
	if strings.Contains(fn, "osx-ssl") {
		return "osx-ssl", nil
	}
	if strings.Contains(fn, "osx") {
		return "osx", nil
	}

	// solaris!
	if strings.Contains(fn, "sunos5") {
		return "sunos5", nil
	}

	return "", errors.Errorf("could not determine platform for %s", fn)
}
