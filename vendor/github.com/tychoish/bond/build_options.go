package bond

import (
	"strings"

	"github.com/pkg/errors"
)

// BuildOptions is a common method to describe a build variant.
type BuildOptions struct {
	Target  string
	Arch    MongoDBArch
	Edition MongoDBEdition
	Debug   bool
}

// Validate checks a BuildOption structure and ensures that there are
// no errors.
func (o BuildOptions) Validate() error {
	var errs []string

	if o.Target == "" {
		errs = append(errs, "target definition is missing")
	}

	if o.Arch == "" {
		errs = append(errs, "arch definition is missing")
	}

	if o.Edition == "" {
		errs = append(errs, "edition definition is missing")
	}

	if len(errs) != 0 {
		return errors.Errorf("%d errors: [%s]",
			len(errs), strings.Join(errs, "; "))
	}

	return nil
}
