// +build windows

package greenbay

import (
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/send"
)

func setupSyslogLogging() send.Sender {
	grip.Warning("syslog is not supported on this platform, falling back to stdout logging.")
	return send.MakeNative()
}
