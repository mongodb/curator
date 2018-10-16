package jasper

import (
	"context"
	"time"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
)

// ProcessTrigger describes the way to write cleanup functions for
// processes, which provide ways of adding behavior to processes after
// they complete.
type ProcessTrigger func(ProcessInfo)

// ProcessTriggerSequence is simply a convenience type to simplify
// running more than one triggered operation.
type ProcessTriggerSequence []ProcessTrigger

// Run loops over triggers and calls each of them successively.
func (s ProcessTriggerSequence) Run(info ProcessInfo) {
	for _, trigger := range s {
		trigger(info)
	}
}

func makeOptionsCloseTrigger() ProcessTrigger {
	return func(info ProcessInfo) {
		info.Options.Close()
	}
}

func makeDefaultTrigger(ctx context.Context, m Manager, opts *CreateOptions, parentID string) ProcessTrigger {
	deadline, hasDeadline := ctx.Deadline()
	timeout := time.Until(deadline)

	return func(info ProcessInfo) {
		switch {
		case info.Timeout:
			var (
				newctx context.Context
				cancel context.CancelFunc
			)

			for _, opt := range opts.OnTimeout {
				if hasDeadline {
					newctx, cancel = context.WithTimeout(context.Background(), timeout)
				} else {
					newctx, cancel = context.WithCancel(ctx)
				}

				p, err := m.Create(newctx, opt)
				if err != nil {
					grip.Warning(message.WrapError(err, message.Fields{
						"trigger": "on-timeout",
						"parent":  parentID,
					}))
					cancel()
					continue
				}
				p.Tag(parentID)
				p.RegisterTrigger(ctx, func(_ ProcessInfo) { cancel() })
			}
		case info.Successful:
			for _, opt := range opts.OnSuccess {
				p, err := m.Create(ctx, opt)
				if err != nil {
					grip.Warning(message.WrapError(err, message.Fields{
						"trigger": "on-success",
						"parent":  parentID,
					}))
					continue
				}
				p.Tag(parentID)
			}
		case !info.Successful:
			for _, opt := range opts.OnFailure {
				p, err := m.Create(ctx, opt)
				if err != nil {

					grip.Warning(message.WrapError(err, message.Fields{
						"trigger": "on-failure",
						"parent":  parentID,
					}))
					continue
				}
				p.Tag(parentID)
			}
		}
	}
}
