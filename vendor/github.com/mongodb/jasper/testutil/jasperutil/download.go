package jasperutil

import (
	"runtime"

	"github.com/mongodb/jasper"
	"github.com/tychoish/bond"
)

func ValidMongoDBDownloadOptions() jasper.MongoDBDownloadOptions {
	target := runtime.GOOS
	if target == "darwin" {
		target = "osx"
	}

	edition := "enterprise"
	if target == "linux" {
		edition = "base"
	}

	return jasper.MongoDBDownloadOptions{
		BuildOpts: bond.BuildOptions{
			Target:  target,
			Arch:    bond.MongoDBArch("x86_64"),
			Edition: bond.MongoDBEdition(edition),
			Debug:   false,
		},
		Releases: []string{"4.0-current"},
	}
}
