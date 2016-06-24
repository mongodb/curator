package repobuilder

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/tychoish/grip"
)

// TODO: in the future we may want to add an entry point or method for
// regenerating these pages throughout the tree, as the current
// integration for this function only regenerates pages on a very
// narrow swath (i.e. only the changed repos.)

func (c *RepositoryConfig) BuildIndexPageForDirectory(path, repoName string) error {
	tmpl, err := template.New("index").Parse(c.Templates.Index)
	if err != nil {
		return err
	}

	catcher := grip.NewCatcher()
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			// do the thing.
			index, err := os.Create(filepath.Join(p, "index.html"))
			catcher.Add(err)
			if err != nil {
				return nil
			}
			defer index.Close()

			var contents []string

			err = filepath.Walk(p, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				contents = append(contents, p)
				return nil
			})
			catcher.Add(err)

			err = tmpl.Execute(index, struct {
				Title    string
				RepoName string
				Files    []string
			}{
				Title:    fmt.Sprintf("Index of %s", p),
				RepoName: repoName,
				Files:    contents,
			})
			catcher.Add(err)

			return nil
		}
		return nil
	})
	catcher.Add(err)

	return catcher.Resolve()
}
