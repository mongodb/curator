package bond

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

type ArtifactsFeed struct {
	Versions []*ArtifactVersion

	mutex sync.RWMutex
	table map[string]*ArtifactVersion
	dir   string
	path  string
}

type ArtifactDownload struct {
	Arch    MongoDBArch
	Edition MongoDBEdition
	Target  string
	Archive struct {
		Debug  string `bson:"debug_symbols" json:"debug_symbols" yaml:"debug_symbols"`
		Sha1   string
		Sha256 string
		Url    string
	}
	Msi      string
	Packages []string
}

func (dl ArtifactDownload) GetBuildOptions() BuildOptions {
	opts := BuildOptions{
		Target:  dl.Target,
		Arch:    dl.Arch,
		Edition: dl.Edition,
	}

	return opts
}

const day = time.Hour * 24

func GetArtifactsFeed(path string) (*ArtifactsFeed, error) {
	feed, err := NewArtifactsFeed(path)
	if err != nil {
		return nil, errors.Wrap(err, "problem building feed")
	}

	if err := feed.Populate(day * 2); err != nil {
		return nil, errors.Wrap(err, "problem getting feed data")
	}

	return feed, nil
}

func NewArtifactsFeed(path string) (*ArtifactsFeed, error) {
	f := &ArtifactsFeed{
		table: make(map[string]*ArtifactVersion),
		path:  path,
	}

	if path == "" {
		// no value for feed, let's write it to the tempDir
		tmpDir, err := ioutil.TempDir("", "mongodb-downloads")
		if err != nil {
			return nil, err
		}

		f.dir = tmpDir
		f.path = filepath.Join(tmpDir, "full.json")
	} else if strings.HasSuffix(path, ".json") {
		f.dir = filepath.Dir(path)
		f.path = path
	} else {
		f.dir = path
		f.path = filepath.Join(f.dir, "full.json")
	}

	if stat, err := os.Stat(f.path); !os.IsNotExist(err) && stat.IsDir() {
		// if the thing we think should be the json file
		// exists but isn't a file (i.e. directory,) then this
		// should be an error.
		return nil, errors.Errorf("path %s not a json file  directory", path)
	}

	return f, nil
}

func (feed *ArtifactsFeed) Populate(ttl time.Duration) error {
	data, err := CacheDownload(ttl, "http://downloads.mongodb.org/full.json", feed.path, false)

	if err != nil {
		return errors.Wrap(err, "problem getting feed data")
	}

	if err = feed.Reload(data); err != nil {
		return errors.Wrap(err, "problem reloading feed")
	}

	return nil
}

func (feed *ArtifactsFeed) Reload(data []byte) error {
	// file exists, remove it if it's more than 48 hours old.
	feed.mutex.Lock()
	defer feed.mutex.Unlock()

	err := json.Unmarshal(data, feed)
	if err != nil {
		return errors.Wrap(err, "problem converting data from json")
	}

	// this is a reload rather than a new load, and we shoiuld
	if len(feed.table) > 0 {
		feed.table = make(map[string]*ArtifactVersion)
	}

	for _, version := range feed.Versions {
		feed.table[version.Version] = version
		version.refresh()
	}

	return err
}

func (feed *ArtifactsFeed) GetVersion(release string) (*ArtifactVersion, bool) {
	feed.mutex.RLock()
	defer feed.mutex.RUnlock()

	version, ok := feed.table[release]
	return version, ok
}

func (feed *ArtifactsFeed) GetLatestArchive(series string, options BuildOptions) (string, error) {
	if len(series) != 3 || string(series[1]) != "." {
		return "", errors.Errorf("series '%s' is not a valid version series", series)
	}

	if options.Debug {
		return "", errors.New("debug symbols are not valid for nightly releases")
	}

	version, ok := feed.GetVersion(series + ".0")
	if !ok {
		return "", errors.Errorf("there is no .0 release for series '%s' in the feed", series)
	}

	dl, err := version.GetDownload(options)
	if err != nil {
		return "", errors.Wrapf(err, "problem fetching download information for series '%s'", series)
	}

	// if it's a dev version: then the branch name is in the file
	// name, and we just take the latest from master
	seriesNum, err := strconv.Atoi(string(series[2]))
	if err != nil {
		// this should be unreachable, because invalid
		// versions won't have yielded results from the feed
		// op
		return "", errors.Wrapf(err, "version specification is invalid")
	}

	if seriesNum%2 == 1 {
		return strings.Replace(dl.Archive.Url, version.Version, "latest", -1), nil
	}

	// if it's a stable version we just replace the version with the word latest.
	return strings.Replace(dl.Archive.Url, version.Version, "v"+series+"-latest", -1), nil
}

func (feed *ArtifactsFeed) GetArchives(releases []string, options BuildOptions) (<-chan string, <-chan error) {
	output := make(chan string)
	errOut := make(chan error)

	go func() {
		catcher := grip.NewCatcher()
		for _, rel := range releases {
			// this is a series, have to handle it differently
			if len(rel) == 3 {
				url, err := feed.GetLatestArchive(rel, options)
				if err != nil {
					catcher.Add(err)
					continue
				}
				output <- url
				continue
			}

			version, ok := feed.GetVersion(rel)
			if !ok {
				catcher.Add(errors.Errorf("no version defined for %s", rel))
				continue
			}
			dl, err := version.GetDownload(options)
			if err != nil {
				catcher.Add(err)
				continue
			}

			if options.Debug {
				output <- dl.Archive.Debug
				continue
			}
			output <- dl.Archive.Url
		}
		close(output)
		if catcher.HasErrors() {
			errOut <- catcher.Resolve()
		}
		close(errOut)
	}()

	return output, errOut
}
