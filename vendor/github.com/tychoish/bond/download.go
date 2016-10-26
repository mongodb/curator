package bond

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

func createDirectory(path string) error {
	stat, err := os.Stat(path)

	if os.IsNotExist(err) {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return errors.Wrapf(err, "problem crating directory %s", path)
		}
		grip.Noticeln("created directory:", path)
	} else if !stat.IsDir() {
		return errors.Errorf("%s exists and is not a directory", path)
	}

	return nil
}

// CacheDownload downloads a resource (url) into a file (path); if the
// file already exists CaceheDownlod does not download a new copy of
// the file, unless local file is older than the ttl, or the force
// option is specified. CacheDownload returns the contents of the file.
func CacheDownload(ttl time.Duration, url, path string, force bool) ([]byte, error) {
	if ttl == 0 {
		force = true
	}

	if stat, err := os.Stat(path); !os.IsNotExist(err) {
		age := time.Since(stat.ModTime())

		if (ttl > 0 && age > ttl) || force {
			grip.Infof("removing stale (%s) file (%s)", age, path)
			if err = os.Remove(path); err != nil {
				return nil, errors.Wrap(err, "problem removing stale feed.")
			}
		}
	}

	// TODO: we're effectively reading the file into memory twice
	// to write it to disk and read it out again.

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := DownloadFile(url, path)
		if err != nil {
			return nil, err
		}
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// DownloadFile downloads a resource (url) into a file specified by
// fileName. Also creates enclosing directories as needed.
func DownloadFile(url, fileName string) error {
	if err := createDirectory(filepath.Dir(fileName)); err != nil {
		return errors.Wrapf(err, "problem creating enclosing directory for %s", fileName)
	}

	if _, err := os.Stat(fileName); !os.IsNotExist(err) {
		return errors.Errorf("'%s' file exists", fileName)
	}

	output, err := os.Create(fileName)
	if err != nil {
		return errors.Wrapf(err, "could not create file for package '%s'", fileName)
	}
	defer output.Close()

	grip.Noticeln("downloading:", fileName)
	response, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "problem downloading file")
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		grip.CatchWarning(os.Remove(fileName))
		return errors.Errorf("encountered error %d (%s) for %s", response.StatusCode, response.Status, url)
	}

	n, err := io.Copy(output, response.Body)
	if err != nil {
		grip.CatchWarning(os.Remove(fileName))

		return errors.Wrapf(err, "problem writing %s to file %s", url, fileName)
	}

	grip.Debugf("%d bytes downloaded. (%s)", n, fileName)
	return nil
}
