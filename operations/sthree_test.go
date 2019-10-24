package operations

import (
	"os"
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

	s.Len(names, 2)
	s.Len(flags, 2)
	s.True(names["file"])
	s.True(names["name"])
}

func (s *CommandsSuite) TestSyncFlagsFactory() {
	pwd, _ := os.Getwd()

	flags := s3syncFlags()
	names := make(map[string]bool)
	for _, flag := range flags {
		flagName := flag.GetName()
		names[flagName] = true
		if flagName == "local" {
			f, ok := flag.(cli.StringFlag)
			s.True(ok)
			s.Equal(pwd, f.Value)
		} else if flagName == "dry-run" || flagName == "delete" || flagName == "serialize" {
			s.IsType(cli.BoolFlag{}, flag)
		} else if flagName == "timeout" {
			s.IsType(cli.DurationFlag{}, flag)
		} else if flagName == "workers" {
			s.IsType(cli.IntFlag{}, flag)
		} else {
			s.IsType(cli.StringFlag{}, flag)
		}
	}

	s.Len(names, 6)
	s.Len(flags, 6)
	s.True(names["local"])
	s.True(names["prefix"])
	s.True(names["delete"])
	s.True(names["timeout"])
	s.True(names["serialize"])
	s.True(names["workers"])
}

func (s *CommandsSuite) TestS3ParentCommandHasExpectedProperties() {
	cmd := S3()
	names := make(map[string]bool)

	for _, sub := range cmd.Subcommands {
		s.IsType(cli.Command{}, sub)
		names[sub.Name] = true

		if sub.Name == "put" {
			s.Equal(sub.Flags, baseS3Flags(s3opFlags(s3putFlags()...)...))
		} else if sub.Name == "get" {
			s.Equal(sub.Flags, baseS3Flags(s3opFlags()...))
		} else if strings.HasPrefix(sub.Name, "sync") {
			s.Equal(sub.Flags, baseS3Flags(s3syncFlags()...))
		}
	}

	s.Len(cmd.Subcommands, 7)
	s.Equal(cmd.Name, "s3")
	s.Len(cmd.Aliases, 1)

	s.True(names["put"])
	s.True(names["get"])
	s.True(names["delete"])
	s.True(names["delete-prefix"])
	s.True(names["sync-to"])
	s.True(names["sync-from"])
}
