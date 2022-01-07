//go:build !linux
// +build !linux

package greenbay

import (
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/send"
)

func setupSystemdLogging() (send.Sender, error) {
	grip.Warning("systemd logging is not supported on this platform, falling back to stdout logging.")
	return send.MakeNative(), nil
}
