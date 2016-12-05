// +build !linux

package grip

import "errors"

// UseSystemdLogger returns an error on non-linux platforms.
func UseSystemdLogger() error {
	return errors.New("systemd not support on this platform")
}
