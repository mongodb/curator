// +build darwin linux

package jasper

import "syscall"

func makeMongodShutdownSignalTrigger() SignalTrigger {
	return func(_ ProcessInfo, _ syscall.Signal) bool {
		return false
	}
}

func makeCleanTerminationSignalTrigger() SignalTrigger {
	return func(_ ProcessInfo, _ syscall.Signal) bool {
		return false
	}
}
