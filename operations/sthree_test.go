package operations

import (
	"os"
	"runtime"
	"strings"

	"github.com/urfave/cli"
)

func (s *CommandsSuite) TestPutGetFlagFactory() {
	flags := s3opFlags()

	names := make(map[string]bool)
	for _, flag := range flags {
		s.IsType(cli.StringFlag{}, flag)
		names[flag.GetName()] = true
	}

	s.Len(names, 4)
	s.Len(flags, 4)
	s.True(names["file"])
	s.True(names["bucket"])
	s.True(names["name"])
}

func (s *CommandsSuite) TestSyncFlagsFactory() {
	pwd, _ := os.Getwd()

	flags := s3syncFlags()
	names := make(map[string]bool)
	for _, flag := range flags {
		names[flag.GetName()] = true
		if flag.GetName() == "local" {
			f, ok := flag.(cli.StringFlag)
			s.True(ok)
			s.Equal(pwd, f.Value)
		} else if flag.GetName() == "jobs" {
			f, ok := flag.(cli.IntFlag)
			s.True(ok)
			s.Equal(f.Value, runtime.NumCPU()*2)
		} else {
			s.IsType(cli.StringFlag{}, flag)
		}
	}

	s.Len(names, 4)
	s.Len(flags, 4)
	s.True(names["bucket"])
	s.True(names["local"])
	s.True(names["prefix"])
}

func (s *CommandsSuite) TestS3ParentCommandHasExpectedProperties() {
	cmd := S3()
	names := make(map[string]bool)

	for _, sub := range cmd.Subcommands {
		s.IsType(cli.Command{}, sub)
		names[sub.Name] = true

		if sub.Name == "put" || sub.Name == "get" {
			s.Equal(sub.Flags, s3opFlags())
		} else if strings.HasPrefix(sub.Name, "sync") {
			s.Equal(sub.Flags, s3syncFlags())
		}
	}

	s.Len(cmd.Subcommands, 6)
	s.Equal(cmd.Name, "s3")
	s.Len(cmd.Aliases, 1)

	s.True(names["put"])
	s.True(names["get"])
	s.True(names["delete"])
	s.True(names["delete-prefix"])
	s.True(names["sync-to"])
	s.True(names["sync-from"])
}
