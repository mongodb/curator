package jasper

import (
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
)

const (
	// The name for the mongod termination event has the format "Global\Mongo_<pid>".
	mongodShutdownEventNamePrefix = "Global\\Mongo_"
	mongodShutdownEventTimeout    = 60 * time.Second

	mongodShutdownSignalTriggerSource   = "mongod shutdown trigger"
	cleanTerminationSignalTriggerSource = "clean termination trigger"
)

func mongodExited(err error) bool {
	return err == ERROR_FILE_NOT_FOUND || err == ERROR_ACCESS_DENIED || err == ERROR_INVALID_HANDLE
}

// makeMongodEventTrigger is necessary to support clean termination of mongods on Windows by signaling the mongod
// shutdown event and waiting for the process to return.
func makeMongodShutdownSignalTrigger() SignalTrigger {
	return func(info ProcessInfo, sig syscall.Signal) bool {
		// Only run if the program is mongod.
		if len(info.Options.Args) == 0 || !strings.Contains(info.Options.Args[0], "mongod") {
			return false
		}
		if sig != syscall.SIGTERM {
			return false
		}

		proc, err := OpenProcess(SYNCHRONIZE, false, uint32(info.PID))
		if err != nil {
			// OpenProcess returns ERROR_INVALID_PARAMETER if the process has already exited.
			if err == ERROR_INVALID_PARAMETER {
				grip.Debug(message.WrapError(err, message.Fields{
					"id":      info.ID,
					"pid":     info.PID,
					"source":  mongodShutdownSignalTriggerSource,
					"message": "did not open process because it has already exited",
				}))
			} else {
				grip.Error(message.WrapError(err, message.Fields{
					"id":      info.ID,
					"pid":     info.PID,
					"source":  mongodShutdownSignalTriggerSource,
					"message": "failed to open process",
				}))
			}
			return false
		}
		defer CloseHandle(proc)

		eventName := mongodShutdownEventNamePrefix + strconv.Itoa(info.PID)
		utf16EventName, err := syscall.UTF16PtrFromString(eventName)
		if err != nil {
			grip.Error(message.WrapError(err, message.Fields{
				"id":      info.ID,
				"pid":     info.PID,
				"source":  mongodShutdownSignalTriggerSource,
				"event":   eventName,
				"message": "failed to convert event name",
			}))
			return false
		}

		event, err := OpenEvent(utf16EventName)
		if err != nil {
			if mongodExited(err) {
				grip.Debug(message.WrapError(err, message.Fields{
					"id":      info.ID,
					"pid":     info.PID,
					"source":  mongodShutdownSignalTriggerSource,
					"event":   eventName,
					"message": "did not open event because process has already exited",
				}))
			} else {
				grip.Error(message.WrapError(err, message.Fields{
					"id":      info.ID,
					"pid":     info.PID,
					"source":  mongodShutdownSignalTriggerSource,
					"event":   eventName,
					"message": "failed to open event",
				}))
			}
			return false
		}
		defer CloseHandle(event)

		if err := SetEvent(event); err != nil {
			if mongodExited(err) {
				grip.Debug(message.WrapError(err, message.Fields{
					"id":      info.ID,
					"pid":     info.PID,
					"source":  mongodShutdownSignalTriggerSource,
					"event":   eventName,
					"message": "did not signal event because process has already exited",
				}))
			} else {
				grip.Error(message.WrapError(err, message.Fields{
					"id":      info.ID,
					"pid":     info.PID,
					"source":  mongodShutdownSignalTriggerSource,
					"event":   eventName,
					"message": "failed to signal event",
				}))
			}
			return false
		}

		waitStatus, err := WaitForSingleObject(proc, mongodShutdownEventTimeout)
		if err != nil {
			grip.Error(message.WrapError(err, message.Fields{
				"id":      info.ID,
				"message": "failed to wait on process",
				"source":  mongodShutdownSignalTriggerSource,
			}))
			return false
		}
		if waitStatus != WAIT_OBJECT_0 {
			grip.Error(message.WrapError(getWaitStatusError(waitStatus), message.Fields{
				"id":      info.ID,
				"message": "wait did not return success",
				"source":  mongodShutdownSignalTriggerSource,
			}))
			return false
		}

		return true
	}
}

// makeCleanTerminationSignalTrigger terminates a process so that it will return exit code 0.
func makeCleanTerminationSignalTrigger() SignalTrigger {
	return func(info ProcessInfo, sig syscall.Signal) bool {
		if sig != syscall.SIGTERM {
			return false
		}

		proc, err := OpenProcess(PROCESS_TERMINATE|PROCESS_QUERY_INFORMATION, false, uint32(info.PID))
		if err != nil {
			// OpenProcess returns ERROR_INVALID_PARAMETER if the process has already exited.
			if err == ERROR_INVALID_PARAMETER {
				grip.Debug(message.WrapError(err, message.Fields{
					"id":      info.ID,
					"pid":     info.PID,
					"source":  cleanTerminationSignalTriggerSource,
					"message": "did not open process because it has already exited",
				}))
			} else {
				grip.Error(message.WrapError(err, message.Fields{
					"id":      info.ID,
					"pid":     info.PID,
					"source":  cleanTerminationSignalTriggerSource,
					"message": "failed to open process",
				}))
			}
			return false
		}
		defer CloseHandle(proc)

		if err := TerminateProcess(proc, 0); err != nil {
			// TerminateProcess returns ERROR_ACCESS_DENIED if the process has already died.
			if err != ERROR_ACCESS_DENIED {
				grip.Error(message.WrapError(err, message.Fields{
					"id":      info.ID,
					"pid":     info.PID,
					"source":  cleanTerminationSignalTriggerSource,
					"message": "failed to terminate process",
				}))
				return false
			}

			var exitCode uint32
			err := GetExitCodeProcess(proc, &exitCode)
			if err != nil {
				grip.Error(message.WrapError(err, message.Fields{
					"id":      info.ID,
					"pid":     info.PID,
					"source":  cleanTerminationSignalTriggerSource,
					"message": "terminate process was sent but failed to get exit code",
				}))
				return false
			}
			if exitCode == STILL_ACTIVE {
				grip.Error(message.WrapError(err, message.Fields{
					"id":      info.ID,
					"pid":     info.PID,
					"source":  cleanTerminationSignalTriggerSource,
					"message": "terminate process was sent but process is still active",
				}))
				return false
			}
		}

		return true
	}
}
