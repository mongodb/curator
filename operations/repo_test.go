package operations

import (
	"github.com/urfave/cli"
)

func (s *CommandsSuite) TestRepoFlags() {
	flags := repoFlags()

	names := make(map[string]bool)
	for _, flag := range flags {
		names[flag.GetName()] = true

		name := flag.GetName()
		if name == "dry-run" || name == "verbose" || name == "rebuild" {
			s.IsType(cli.BoolFlag{}, flag)
		} else if name == "timeout" {
			s.IsType(cli.DurationFlag{}, flag)
		} else if name == "retries" {
			s.IsType(cli.IntFlag{}, flag)
		} else {
			s.IsType(cli.StringFlag{}, flag)
		}
	}

	s.Len(names, 7)
	s.Len(flags, 7)
	s.True(names["config"])
	s.True(names["distro"])
	s.True(names["version"])
	s.True(names["edition"])
	s.True(names["timeout"])
	s.True(names["arch"])
	s.True(names["profile"])
}
