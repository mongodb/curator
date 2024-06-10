package rollups

import (
	"sort"

	"github.com/aclements/go-moremath/stats"
)

type MetricType string

const (
	MetricTypeMean         MetricType = "mean"
	MetricTypeMedian       MetricType = "median"
	MetricTypeMax          MetricType = "max"
	MetricTypeMin          MetricType = "min"
	MetricTypeSum          MetricType = "sum"
	MetricTypeStdDev       MetricType = "standard-deviation"
	MetricTypePercentile99 MetricType = "percentile-99th"
	MetricTypePercentile90 MetricType = "percentile-90th"
	MetricTypePercentile95 MetricType = "percentile-95th"
	MetricTypePercentile80 MetricType = "percentile-80th"
	MetricTypePercentile50 MetricType = "percentile-50th"
	MetricTypeThroughput   MetricType = "throughput"
	MetricTypeLatency      MetricType = "latency"
)

//////////////////
// Default Means
//////////////////

type latencyAverage struct{}

const (
	latencyAverageName    = "AverageLatency"
	latencyAverageVersion = 3
)

func (f *latencyAverage) Type() string    { return latencyAverageName }
func (f *latencyAverage) Names() []string { return []string{latencyAverageName} }
func (f *latencyAverage) Version() int    { return latencyAverageVersion }
func (f *latencyAverage) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	rollup := PerfRollupValue{
		Name:          latencyAverageName,
		Version:       latencyAverageVersion,
		MetricType:    MetricTypeMean,
		UserSubmitted: user,
	}

	if s.counters.operationsTotal > 0 {
		rollup.Value = float64(s.timers.durationTotal) / float64(s.counters.operationsTotal)
	}

	return []PerfRollupValue{rollup}
}

type sizeAverage struct{}

const (
	sizeAverageName    = "AverageSize"
	sizeAverageVersion = 3
)

func (f *sizeAverage) Type() string    { return sizeAverageName }
func (f *sizeAverage) Names() []string { return []string{sizeAverageName} }
func (f *sizeAverage) Version() int    { return sizeAverageVersion }
func (f *sizeAverage) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	rollup := PerfRollupValue{
		Name:          sizeAverageName,
		Version:       sizeAverageVersion,
		MetricType:    MetricTypeMean,
		UserSubmitted: user,
	}

	if s.counters.operationsTotal > 0 {
		rollup.Value = float64(s.counters.sizeTotal) / float64(s.counters.operationsTotal)
	}

	return []PerfRollupValue{rollup}
}

////////////////////////
// Default Throughputs
////////////////////////

type operationThroughput struct{}

const (
	operationThroughputName    = "OperationThroughput"
	operationThroughputVersion = 5
)

func (f *operationThroughput) Type() string    { return operationThroughputName }
func (f *operationThroughput) Names() []string { return []string{operationThroughputName} }
func (f *operationThroughput) Version() int    { return operationThroughputVersion }
func (f *operationThroughput) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	rollup := PerfRollupValue{
		Name:          operationThroughputName,
		Version:       operationThroughputVersion,
		MetricType:    MetricTypeThroughput,
		UserSubmitted: user,
	}

	if s.timers.totalWallTime > 0 {
		rollup.Value = float64(s.counters.operationsTotal) / s.timers.totalWallTime.Seconds()
	}

	return []PerfRollupValue{rollup}
}

type documentThroughput struct{}

const (
	documentThroughputName    = "DocumentThroughput"
	documentThroughputVersion = 1
)

func (f *documentThroughput) Type() string    { return documentThroughputName }
func (f *documentThroughput) Names() []string { return []string{documentThroughputName} }
func (f *documentThroughput) Version() int    { return documentThroughputVersion }
func (f *documentThroughput) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	rollup := PerfRollupValue{
		Name:          documentThroughputName,
		Version:       documentThroughputVersion,
		MetricType:    MetricTypeThroughput,
		UserSubmitted: user,
	}

	if s.timers.totalWallTime > 0 {
		rollup.Value = float64(s.counters.documentsTotal) / s.timers.totalWallTime.Seconds()
	}

	return []PerfRollupValue{rollup}
}

type sizeThroughput struct{}

const (
	sizeThroughputName    = "SizeThroughput"
	sizeThroughputVersion = 5
)

func (f *sizeThroughput) Type() string    { return sizeThroughputName }
func (f *sizeThroughput) Names() []string { return []string{sizeThroughputName} }
func (f *sizeThroughput) Version() int    { return sizeThroughputVersion }
func (f *sizeThroughput) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	rollup := PerfRollupValue{
		Name:          sizeThroughputName,
		Version:       sizeThroughputVersion,
		MetricType:    MetricTypeThroughput,
		UserSubmitted: user,
	}
	if s.timers.totalWallTime > 0 {
		rollup.Value = float64(s.counters.sizeTotal) / s.timers.totalWallTime.Seconds()
	}

	return []PerfRollupValue{rollup}
}

type errorThroughput struct{}

const (
	errorThroughputName    = "ErrorRate"
	errorThroughputVersion = 5
)

func (f *errorThroughput) Type() string    { return errorThroughputName }
func (f *errorThroughput) Names() []string { return []string{errorThroughputName} }
func (f *errorThroughput) Version() int    { return errorThroughputVersion }
func (f *errorThroughput) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	rollup := PerfRollupValue{
		Name:          errorThroughputName,
		Version:       errorThroughputVersion,
		MetricType:    MetricTypeThroughput,
		UserSubmitted: user,
	}

	if s.timers.totalWallTime > 0 {
		rollup.Value = float64(s.counters.errorsTotal) / s.timers.totalWallTime.Seconds()
	}

	return []PerfRollupValue{rollup}
}

////////////////////////
// Default Percentiles
////////////////////////

type latencyPercentile struct{}

const (
	latencyPercentileName    = "LatencyPercentile"
	latencyPercentile50Name  = "Latency50thPercentile"
	latencyPercentile80Name  = "Latency80thPercentile"
	latencyPercentile90Name  = "Latency90thPercentile"
	latencyPercentile95Name  = "Latency95thPercentile"
	latencyPercentile99Name  = "Latency99thPercentile"
	latencyPercentileVersion = 4
)

func (f *latencyPercentile) Type() string { return latencyPercentileName }
func (f *latencyPercentile) Names() []string {
	return []string{
		latencyPercentile50Name,
		latencyPercentile80Name,
		latencyPercentile90Name,
		latencyPercentile95Name,
		latencyPercentile99Name,
	}
}
func (f *latencyPercentile) Version() int { return latencyPercentileVersion }
func (f *latencyPercentile) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	p50 := PerfRollupValue{
		Name:          latencyPercentile50Name,
		Version:       latencyPercentileVersion,
		MetricType:    MetricTypePercentile50,
		UserSubmitted: user,
	}
	p80 := PerfRollupValue{
		Name:          latencyPercentile80Name,
		Version:       latencyPercentileVersion,
		MetricType:    MetricTypePercentile80,
		UserSubmitted: user,
	}
	p90 := PerfRollupValue{
		Name:          latencyPercentile90Name,
		Version:       latencyPercentileVersion,
		MetricType:    MetricTypePercentile90,
		UserSubmitted: user,
	}
	p95 := PerfRollupValue{
		Name:          latencyPercentile95Name,
		Version:       latencyPercentileVersion,
		MetricType:    MetricTypePercentile95,
		UserSubmitted: user,
	}
	p99 := PerfRollupValue{
		Name:          latencyPercentile99Name,
		Version:       latencyPercentileVersion,
		MetricType:    MetricTypePercentile99,
		UserSubmitted: user,
	}

	if len(s.timers.extractedDurations) > 0 {
		durs := make(sort.Float64Slice, len(s.timers.extractedDurations))
		copy(durs, s.timers.extractedDurations)
		durs.Sort()
		latencySample := stats.Sample{
			Xs:     durs,
			Sorted: true,
		}
		p50.Value = latencySample.Quantile(0.5)
		p80.Value = latencySample.Quantile(0.8)
		p90.Value = latencySample.Quantile(0.9)
		p95.Value = latencySample.Quantile(0.95)
		p99.Value = latencySample.Quantile(0.99)
	}

	return []PerfRollupValue{p50, p80, p90, p95, p99}
}

///////////////////
// Default Bounds
///////////////////

type workersBounds struct{}

const (
	workersBoundsName    = "WorkersBounds"
	workersMinName       = "WorkersMin"
	workersMaxName       = "WorkersMax"
	workersBoundsVersion = 3
)

func (f *workersBounds) Type() string    { return workersBoundsName }
func (f *workersBounds) Names() []string { return []string{workersMinName, workersMaxName} }
func (f *workersBounds) Version() int    { return workersBoundsVersion }
func (f *workersBounds) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	min := PerfRollupValue{
		Name:          workersMinName,
		Version:       workersBoundsVersion,
		MetricType:    MetricTypeMin,
		UserSubmitted: user,
	}
	max := PerfRollupValue{
		Name:          workersMaxName,
		Version:       workersBoundsVersion,
		MetricType:    MetricTypeMax,
		UserSubmitted: user,
	}
	if len(s.gauges.workers) > 0 {
		min.Value, max.Value = stats.Sample{Xs: s.gauges.workers}.Bounds()
	}

	return []PerfRollupValue{min, max}
}

type latencyBounds struct{}

const (
	latencyBoundsName    = "LatencyBounds"
	latencyMinName       = "LatencyMin"
	latencyMaxName       = "LatencyMax"
	latencyBoundsVersion = 4
)

func (f *latencyBounds) Type() string    { return latencyBoundsName }
func (f *latencyBounds) Names() []string { return []string{latencyMinName, latencyMaxName} }
func (f *latencyBounds) Version() int    { return latencyBoundsVersion }
func (f *latencyBounds) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	min := PerfRollupValue{
		Name:          latencyMinName,
		Version:       latencyBoundsVersion,
		MetricType:    MetricTypeMin,
		UserSubmitted: user,
	}
	max := PerfRollupValue{
		Name:          latencyMaxName,
		Version:       latencyBoundsVersion,
		MetricType:    MetricTypeMax,
		UserSubmitted: user,
	}

	if len(s.timers.extractedDurations) > 0 {
		min.Value, max.Value = stats.Sample{Xs: s.timers.extractedDurations}.Bounds()
	}

	return []PerfRollupValue{min, max}
}

/////////////////
// Default Sums
/////////////////

type durationSum struct{}

const (
	durationSumName    = "DurationTotal"
	durationSumVersion = 5
)

func (f *durationSum) Type() string    { return durationSumName }
func (f *durationSum) Names() []string { return []string{durationSumName} }
func (f *durationSum) Version() int    { return durationSumVersion }
func (f *durationSum) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	return []PerfRollupValue{
		{
			Name:          durationSumName,
			Value:         s.timers.totalWallTime,
			Version:       durationSumVersion,
			MetricType:    MetricTypeSum,
			UserSubmitted: user,
		},
	}
}

type errorsSum struct{}

const (
	errorsSumName    = "ErrorsTotal"
	errorsSumVersion = 3
)

func (f *errorsSum) Type() string    { return errorsSumName }
func (f *errorsSum) Names() []string { return []string{errorsSumName} }
func (f *errorsSum) Version() int    { return errorsSumVersion }
func (f *errorsSum) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	return []PerfRollupValue{
		{
			Name:          errorsSumName,
			Value:         s.counters.errorsTotal,
			Version:       errorsSumVersion,
			MetricType:    MetricTypeSum,
			UserSubmitted: user,
		},
	}
}

type operationsSum struct{}

const (
	operationsSumName    = "OperationsTotal"
	operationsSumVersion = 3
)

func (f *operationsSum) Type() string    { return operationsSumName }
func (f *operationsSum) Names() []string { return []string{operationsSumName} }
func (f *operationsSum) Version() int    { return operationsSumVersion }
func (f *operationsSum) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	return []PerfRollupValue{
		{
			Name:          operationsSumName,
			Value:         s.counters.operationsTotal,
			Version:       operationsSumVersion,
			MetricType:    MetricTypeSum,
			UserSubmitted: user,
		},
	}
}

type documentsSum struct{}

const (
	documentsSumName    = "DocumentsTotal"
	documentsSumVersion = 0
)

func (f *documentsSum) Type() string    { return documentsSumName }
func (f *documentsSum) Names() []string { return []string{documentsSumName} }
func (f *documentsSum) Version() int    { return documentsSumVersion }
func (f *documentsSum) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	return []PerfRollupValue{
		{
			Name:          documentsSumName,
			Value:         s.counters.documentsTotal,
			Version:       documentsSumVersion,
			MetricType:    MetricTypeSum,
			UserSubmitted: user,
		},
	}
}

type sizeSum struct{}

const (
	sizeSumName    = "SizeTotal"
	sizeSumVersion = 3
)

func (f *sizeSum) Type() string    { return sizeSumName }
func (f *sizeSum) Names() []string { return []string{sizeSumName} }
func (f *sizeSum) Version() int    { return sizeSumVersion }
func (f *sizeSum) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	return []PerfRollupValue{
		{
			Name:          sizeSumName,
			Value:         s.counters.sizeTotal,
			Version:       sizeSumVersion,
			MetricType:    MetricTypeSum,
			UserSubmitted: user,
		},
	}
}

type overheadSum struct{}

const (
	overheadSumName    = "OverheadTotal"
	overheadSumVersion = 1
)

func (f *overheadSum) Type() string    { return overheadSumName }
func (f *overheadSum) Names() []string { return []string{overheadSumName} }
func (f *overheadSum) Version() int    { return overheadSumVersion }
func (f *overheadSum) Calc(s *PerformanceStatistics, user bool) []PerfRollupValue {
	return []PerfRollupValue{
		{
			Name:          overheadSumName,
			Value:         s.timers.total - s.timers.durationTotal,
			Version:       overheadSumVersion,
			MetricType:    MetricTypeSum,
			UserSubmitted: user,
		},
	}
}
