package internal

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/mongodb/ftdc/events"
)

func (m *EventMetrics) Export() *events.Performance {
	dur, _ := ptypes.Duration(m.Timers.Duration)
	total, _ := ptypes.Duration(m.Timers.Total)

	return &events.Performance{
		ID: m.Id,
		Counters: events.PerformanceCounters{
			Number:     m.Counters.Number,
			Operations: m.Counters.Ops,
			Size:       m.Counters.Size,
			Errors:     m.Counters.Errors,
		},
		Timers: events.PerformanceTimers{
			Duration: dur,
			Total:    total,
		},
		Gauges: events.PerformanceGauges{
			State:   m.Gauges.State,
			Workers: m.Gauges.Workers,
			Failed:  m.Gauges.Failed,
		},
	}
}
