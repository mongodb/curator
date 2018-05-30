// +build linux

package greenbay

import "github.com/mongodb/grip/send"

func setupSystemdLogging() (send.Sender, error) {
	return send.MakeSystemdLogger()
}
