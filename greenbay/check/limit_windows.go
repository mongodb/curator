//go:build windows
// +build windows

package check

import (
	"github.com/pkg/errors"
	"golang.org/x/sys/windows/registry"
)

// because the limit_*.go files are only built on some platforms, we
// define a "limitValueCheckTable" function in all files which returns
// a map of "limit name" to check function.

func limitValueCheckTable() map[string]limitValueCheck {
	m := map[string]limitValueCheck{
		"irp-stack-size": irpStackSize,
	}

	// we have to define UNIX tests here as "invalid checks" so
	// windows and unix systems can use the same config
	for _, name := range []string{"open-files", "address-size"} {
		m[name] = undefinedLimitCheckFactory(name)
	}

	return m
}

func irpStackSize(value int) (bool, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\services\LanmanServer\Parameters`, registry.QUERY_VALUE)
	if err != nil {
		return false, errors.Wrap(err, "problem opening registry key")
	}
	defer key.Close()

	irpStackSize, _, err := key.GetIntegerValue("IRPStackSize")
	if err != nil {
		return false, errors.Wrap(err, "problem getting value of IRPStackSize Value")
	}

	if irpStackSize != uint64(value) {
		return false, errors.Errorf("IRPStackSize should be %d but is %d", value, irpStackSize)
	}

	return true, nil

}
