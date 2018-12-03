package internal

import "github.com/evergreen-ci/poplar"

func (rt CreateOptions_RecorderType) Export() poplar.RecorderType {
	switch rt {
	case CreateOptions_PERF:
		return poplar.RecorderPerf
	case CreateOptions_PERF_SINGLE:
		return poplar.RecorderPerfSingle
	case CreateOptions_PERF_100MS:
		return poplar.RecorderPerf100ms
	case CreateOptions_PERF_1S:
		return poplar.RecorderPerf1s
	case CreateOptions_HISTOGRAM_SINGLE:
		return poplar.RecorderHistogramSingle
	case CreateOptions_HISTOGRAM_100MS:
		return poplar.RecorderHistogram100ms
	case CreateOptions_HISTOGRAM_1S:
		return poplar.RecorderHistogram1s
	case CreateOptions_INTERVAL_SUMMARIZATION:
		return poplar.CustomMetrics
	default:
		return ""
	}
}

func (opts *CreateOptions) Export() poplar.CreateOptions {
	return poplar.CreateOptions{
		Path:      opts.Path,
		ChunkSize: int(opts.ChunkSize),
		Streaming: opts.Streaming,
		Dynamic:   opts.Dynamic,
		Recorder:  opts.Recorder.Export(),
	}
}
