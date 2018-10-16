package jasper

import "github.com/pkg/errors"

type Filter string

const (
	Running    Filter = "running"
	Terminated        = "terminated"
	All               = "all"
	Failed            = "failed"
	Successful        = "successful"
)

func (f Filter) Validate() error {
	switch f {
	case Running, Terminated, All, Failed, Successful:
		return nil
	default:
		return errors.Errorf("%s is not a valid filter", f)
	}
}
