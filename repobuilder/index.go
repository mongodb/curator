package repobuilder

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
)

// TODO: in the future we may want to add an entry point or method for
// regenerating these pages throughout the tree, as the current
// integration for this function only regenerates pages on a very
// narrow swath (i.e. only the changed repos.)

// BuildIndexPageForDirectory builds default Apache HTTPD-style
// directory listing index.html files for a hierarchy.
func (c *RepositoryConfig) BuildIndexPageForDirectory(path, repoName string) error {
	tmpl, err := template.New("index").Parse(c.Templates.Index)
	if err != nil {
		return errors.Wrap(err, "problem parsing index file template")
	}

	catcher := grip.NewCatcher()
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// we want to write once index.html per directory. If
		// we don't have directory, we can't do anything here.
		if info.IsDir() {
			// figure out the contents of the directory
			var contents []string
			numDirs := getNumDirs(p)

			err = filepath.Walk(p, func(contentPath string, info os.FileInfo, err error) error {
				// for each directory we walk its contents and add things to the listing for
				// that directory. This is not an optimal algorithm.

				if err != nil {
					return err
				}

				// skip listing "self"
				if contentPath == p {
					return nil
				}

				// don't list index.html files
				if strings.HasSuffix(contentPath, "index.html") {
					return nil
				}

				// we want to avoid list things recursively on each page. instead we only things if
				// it has one more element (i.e. a file name or sub directory) than the enclosing directory.
				if getNumDirs(contentPath)-1 == numDirs {
					contents = append(contents, filepath.Base(contentPath))
				}

				return nil
			})
			catcher.Add(err)

			// build content and write it to file
			buffer := bytes.NewBuffer([]byte{})

			err = tmpl.Execute(buffer, struct {
				Title    string
				RepoName string
				Files    []string
			}{
				Title:    fmt.Sprintf("Index of %s", filepath.Base(p)),
				RepoName: repoName,
				Files:    contents,
			})
			catcher.Add(err)
			if err != nil {
				return nil
			}

			err = ioutil.WriteFile(filepath.Join(p, "index.html"), buffer.Bytes(), 0644)
			catcher.Add(err)
			if err != nil {
				return nil
			}

			grip.Debug(message.Fields{
				"message": "wrote index file",
				"path":    filepath.Join(p, "index.html"),
				"repo":    repoName,
			})
			return nil
		}
		return nil
	})
	catcher.Add(err)

	return catcher.Resolve()
}

func getNumDirs(path string) int {
	return len(strings.Split(path, string(os.PathSeparator)))
}
