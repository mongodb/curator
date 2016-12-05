// +build !linux

package logging

import "errors"

// UseSystemdLogger returns an error on non-linux platforms.
func (g *Grip) UseSystemdLogger() error {
	return errors.New("systemd not support on this platform")
}
