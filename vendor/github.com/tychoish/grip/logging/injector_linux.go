// +build linux
package logging

import "github.com/tychoish/grip/send"

// UseSystemdLogger set the Journaler to use the systemd loggerwithout
// changing the configuration of the Journaler.
func (g *Grip) UseSystemdLogger() error {
	// name, threshold, default
	sender, err := send.NewJournaldLogger(g.name, g.sender.ThresholdLevel(), g.sender.DefaultLevel())
	if err != nil {
		if g.Sender().Name() == "bootstrap" {
			g.SetSender(sender)
		}
		return err
	}
	g.SetSender(sender)
	return nil
}
