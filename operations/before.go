package operations

import (
	"os"

	"github.com/mongodb/grip"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func requireFileExists(name string, hasDefault bool) cli.BeforeFunc {
	return func(c *cli.Context) error {
		path := c.String(name)
		if path == "" {
			if !hasDefault {
				return errors.Errorf("flag '--%s' was not specified", name)
			}
			return nil
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			return errors.Errorf("file '%s' does not exist", path)
		}

		return nil
	}
}

func requireStringFlag(name string) cli.BeforeFunc {
	return func(c *cli.Context) error {
		if c.String(name) == "" {
			return errors.Errorf("flag '--%s' was not specified", name)
		}
		return nil
	}
}

func mergeBeforeFuncs(ops ...func(c *cli.Context) error) cli.BeforeFunc {
	return func(c *cli.Context) error {
		catcher := grip.NewBasicCatcher()

		for _, op := range ops {
			catcher.Add(op(c))
		}

		return catcher.Resolve()
	}
}

func requireFileOrPositional(pathFlagName string) cli.BeforeFunc {
	return func(c *cli.Context) error {
		path := c.String(pathFlagName)
		if path == "" {
			if c.NArg() != 1 {
				return errors.New("must specify a path")
			}
			path = c.Args().Get(0)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			return errors.Errorf("file %s does not exist", path)
		}

		return c.Set(pathFlagName, path)
	}
}
