package greenbay

import (
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/send"
	"github.com/pkg/errors"
)

// SetupLogging is a helper to configure the global grip logging
// instance, and is used in the main package to configure logging for
// the Greebay Service. Reconfigures the logging backend for the
// process' default "grip" logging instance.
func SetupLogging(format string, fileName string) error {
	var sender send.Sender
	var err error

	switch format {
	case "stdout":
		sender = send.MakeNative()
	case "stderr":
		sender = send.MakeErrorLogger()
	case "file":
		sender, err = send.MakeFileLogger(fileName)
	case "json-stdout":
		sender = send.MakeJSONConsoleLogger()
	case "json-file":
		sender, err = send.MakeJSONFileLogger(fileName)
	case "systemd":
		sender, err = setupSystemdLogging()
	case "syslog":
		sender = setupSyslogLogging()
	default:
		grip.Warningf("no supported output format '%s' writing log messages to standard output", format)
		sender = send.MakeNative()
	}

	if err != nil {
		return errors.Wrapf(err, "configuring log type '%s'", format)
	}

	return grip.SetSender(sender)
}
