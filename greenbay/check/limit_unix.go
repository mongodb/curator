//go:build linux || freebsd || solaris || darwin
// +build linux freebsd solaris darwin

package check

import (
	"runtime"
	"syscall"

	"github.com/pkg/errors"
)

// because the limit_*.go files are only built on some platforms, we
// define a "limitValueCheckTable" function in all files which returns
// a map of "limit name" to check function.

func limitValueCheckTable() map[string]limitValueCheck {
	var addressLimit uint64

	if runtime.GOARCH == "386" {
		addressLimit = 2147483600
	} else {
		addressLimit = 18446744073709551000
	}

	return map[string]limitValueCheck{
		"open-files":     limitCheckFactory("open-files", syscall.RLIMIT_NOFILE, 128000),
		"address-size":   limitCheckFactory("address-size", syscall.RLIMIT_AS, addressLimit),
		"irp-stack-size": undefinedLimitCheckFactory("irp-stack-size"),
	}
}

func limitCheckFactory(name string, resource int, max uint64) limitValueCheck {
	return func(value int) (bool, error) {
		limits := &syscall.Rlimit{}

		err := syscall.Getrlimit(resource, limits)
		if err != nil {
			return false, errors.Wrapf(err, "finding '%s' limit", name)
		}

		expected := uint64(value)
		if value < 0 {
			expected = max
		}

		if limits.Max < expected {
			return false, errors.Errorf("'%s' limit is %d which is less than %d",
				name, limits.Max, value)
		}

		return true, nil
	}
}
