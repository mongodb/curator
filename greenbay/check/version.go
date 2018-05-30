package check

import (
	"github.com/blang/semver"
	"github.com/pkg/errors"
)

func compareVersions(rel string, actual, expected semver.Version) (bool, error) {
	switch rel {
	case "gte", "":
		return actual.GTE(expected), nil
	case "lte":
		return actual.LTE(expected), nil
	case "lt":
		return actual.LT(expected), nil
	case "gt":
		return actual.GT(expected), nil
	case "eq":
		return actual.EQ(expected), nil
	default:
		return false, errors.Errorf("relationship '%s' is not valid", rel)
	}
}
