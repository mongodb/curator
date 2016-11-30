package bond

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

// BuildCatalog is a structure that represents a group of MongoDB
// artifacts managed by bond, and provides an interface for retrieving
// artifacts.
type BuildCatalog struct {
	Path  string
	table map[BuildInfo]string
	mutex sync.RWMutex
}

// NewCatalog populates and returns a BuildCatalog object from a given path.
func NewCatalog(path string) (*BuildCatalog, error) {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrap(err, "problem resolving absolute path")
	}

	contents, err := getContents(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not find contents")
	}

	cache := &BuildCatalog{
		Path:  path,
		table: map[BuildInfo]string{},
	}

	catcher := grip.NewCatcher()
	for _, obj := range contents {
		if !obj.IsDir() {
			continue
		}

		if !strings.HasPrefix(obj.Name(), "mongodb-") {
			continue
		}

		fileName := filepath.Join(path, obj.Name())

		if err := cache.Add(fileName); err != nil {
			catcher.Add(err)
			continue
		}
	}

	if catcher.HasErrors() {
		return nil, errors.Wrapf(catcher.Resolve(),
			"problem building build catalog from path: %s", path)
	}

	return cache, nil
}

// Add adds a build to the catalog, and returns an error if it's not a
// valid build. The build file name must be a part of the path
// specified when creating the BuildCatalog object, otherwise Add will
// not add this item to the cache and return an error and .
func (c *BuildCatalog) Add(fileName string) error {
	fileName, err := filepath.Abs(fileName)
	if err != nil {
		return errors.Wrapf(err, "problem finding absolute path of %s object", fileName)
	}

	if !strings.HasPrefix(fileName, c.Path) {
		return errors.Errorf("cannot add %s to '%s' cache because it is not in the same root",
			fileName, c.Path)
	}

	info, err := GetInfoFromFileName(fileName)
	if err != nil {
		return errors.Wrap(err, "problem collecting information about build")
	}

	err = validateBuildArtifacts(fileName, info.Version)
	if err != nil {
		return errors.Wrapf(err, "problem validating contents of %s", fileName)
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if _, ok := c.table[info]; ok {
		return errors.Errorf("path %s exists in cache", fileName)
	}

	c.table[info] = fileName

	return nil
}

// Get returns the path to a build in the BuildCatalog based on the
// parameters presented. Returns an error if a build matching the
// parameters specified does not exist in the cache.
func (c *BuildCatalog) Get(version, edition, target, arch string, debug bool) (string, error) {
	info := BuildInfo{
		Version: version,
		Options: BuildOptions{
			Target:  target,
			Arch:    MongoDBArch(arch),
			Edition: MongoDBEdition(edition),
			Debug:   debug,
		},
	}

	// TODO consider if we want to validate against bad or invalid
	// options; potentially by extending the buildinfo validation method.

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	path, ok := c.table[info]
	if !ok {
		return "", errors.Errorf("could not find version %s, edition %s, target %s, arch %s in %s",
			version, edition, target, arch, c.Path)
	}

	return path, nil
}

func getContents(path string) ([]os.FileInfo, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {

		return []os.FileInfo{}, errors.Errorf("path %s does not exist", path)
	}

	contents, err := ioutil.ReadDir(path)
	if err != nil {
		return []os.FileInfo{}, errors.Wrapf(err, "problem fetching contents of %s", path)
	}

	if len(contents) == 0 {
		return []os.FileInfo{}, errors.Errorf("path %s is empty", path)
	}

	return contents, nil
}

func validateBuildArtifacts(path, version string) error {
	path = filepath.Join(path, "bin")

	contents, err := getContents(path)
	if err != nil {
		return errors.Wrapf(err, "problem finding contents for %s", version)
	}

	pkg := make(map[string]struct{})
	for _, info := range contents {
		pkg[info.Name()] = struct{}{}
	}

	catcher := grip.NewCatcher()
	for _, bin := range []string{"mongod", "mongos"} {
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}

		if _, ok := pkg[bin]; !ok {
			catcher.Add(errors.Errorf("binary %s is missing from %s for %s",
				bin, path, version))
		}
	}

	return catcher.Resolve()
}
