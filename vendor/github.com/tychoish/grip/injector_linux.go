// +build linux

package grip

// UseSystemdLogger configures the standard grip package logger to use
// the systemd loggerwithout changing the configuration of the
// Journaler.
func UseSystemdLogger() error {
	return std.UseSystemdLogger()
}
