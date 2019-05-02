package poplar

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mongodb/ftdc"
	"github.com/mongodb/ftdc/events"
	"github.com/pkg/errors"
)

// RecorderType represents the underlying recorder type.
type RecorderType string

const (
	RecorderPerf            RecorderType = "perf"
	RecorderPerfSingle                   = "perf-single"
	RecorderPerf100ms                    = "perf-grouped-100ms"
	RecorderPerf1s                       = "perf-grouped-1s"
	RecorderHistogramSingle              = "histogram-single"
	RecorderHistogram100ms               = "histogram-grouped-100ms"
	RecorderHistogram1s                  = "histogram-grouped-1s"
	CustomMetrics                        = "custom"
)

// Validate the underyling recorder type.
func (t RecorderType) Validate() error {
	switch t {
	case RecorderPerf, RecorderPerfSingle, RecorderPerf100ms, RecorderPerf1s,
		RecorderHistogramSingle, RecorderHistogram100ms, RecorderHistogram1s, CustomMetrics:

		return nil
	default:
		return errors.Errorf("%s is not a supported recorder type", t)
	}
}

type recorderInstance struct {
	file      io.WriteCloser
	collector ftdc.Collector
	recorder  events.Recorder
	tracker   *customEventTracker
	isDynamic bool
	isCustom  bool
}

type customEventTracker struct {
	events.Custom
	sync.Mutex
}

func (c *customEventTracker) Add(key string, value interface{}) error {
	if c == nil {
		return errors.New("tracker is not populated")
	}

	c.Lock()
	defer c.Unlock()

	return errors.WithStack(c.Custom.Add(key, value))
}

func (c *customEventTracker) Reset() {
	c.Lock()
	defer c.Unlock()

	c.Custom = events.MakeCustom(cap(c.Custom))
}

func (c *customEventTracker) Dump() events.Custom {
	c.Lock()
	defer c.Unlock()

	return c.Custom
}

// CustomMetricsCollector defines an interface for collecting metrics.
type CustomMetricsCollector interface {
	Add(string, interface{}) error
	Dump() events.Custom
	Reset()
}

// RecorderRegistry caches instances of recorders.
type RecorderRegistry struct {
	cache       map[string]*recorderInstance
	benchPrefix string
	mu          sync.Mutex
}

// NewRegistry returns a new (empty) RecorderRegistry.
func NewRegistry() *RecorderRegistry {
	return &RecorderRegistry{
		cache: map[string]*recorderInstance{},
	}
}

// Create builds a new collector, of the given name with the specified
// options controling the collector type and configuration.
//
// If the options specify a filename that already exists, then Create
// will return an error.
func (r *RecorderRegistry) Create(key string, collOpts CreateOptions) (events.Recorder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.cache[key]
	if ok {
		return nil, errors.Errorf("a recorder named '%s' already exists", key)
	}

	instance, err := collOpts.build()
	if err != nil {
		return nil, errors.Wrap(err, "could not construct recorder output")
	}

	r.cache[key] = instance

	return instance.recorder, nil
}

// GetRecorder returns the Recorder instance for this key. Returns
// false when the recorder does not exist.
func (r *RecorderRegistry) GetRecorder(key string) (events.Recorder, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	impl, ok := r.cache[key]
	if !ok {
		return nil, false
	}

	return impl.recorder, true
}

// GetCustomCollector returns the CustomMetricsCollector instance for this key.
// Returns false when the collector does not exist.
func (r *RecorderRegistry) GetCustomCollector(key string) (CustomMetricsCollector, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	impl, ok := r.cache[key]
	if !ok {
		return nil, false
	}

	if !impl.isCustom || impl.tracker == nil {
		return nil, false
	}

	return impl.tracker, true
}

// GetCollector returns the collector instance for this key. Will
// return false, when the collector does not exist OR if the collector
// is dynamic.
func (r *RecorderRegistry) GetCollector(key string) (ftdc.Collector, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	impl, ok := r.cache[key]

	if !ok || !impl.isDynamic {
		return nil, false
	}

	return impl.collector, true
}

// SetBenchRecorderPrefix sets the bench prefix for this registry.
func (r *RecorderRegistry) SetBenchRecorderPrefix(prefix string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.benchPrefix = prefix
}

// MakeBenchmark configures a recorder to support executing a
// BenchmarkCase in the form of a standard library benchmarking
// format.
func (r *RecorderRegistry) MakeBenchmark(bench *BenchmarkCase) func(*testing.B) {
	name := bench.Name()
	r.mu.Lock()
	fqname := filepath.Join(r.benchPrefix, name) + ".ftdc"
	r.mu.Unlock()

	recorder, err := r.Create(name, CreateOptions{
		Path:      fqname,
		ChunkSize: 1024,
		Streaming: true,
		Dynamic:   true,
		Recorder:  bench.Recorder,
	})

	if err != nil {
		return func(b *testing.B) { b.Fatal(errors.Wrap(err, "problem making recorder")) }
	}

	return bench.Bench.standard(recorder, func() error { return r.Close(name) })
}

// Close flushes and closes the underlying recorder and collector and
// then removes it from the cache.
func (r *RecorderRegistry) Close(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if impl, ok := r.cache[key]; ok {
		if impl.isCustom {
			if err := impl.collector.Add(impl.tracker.Custom); err != nil {
				return errors.Wrap(err, "problem flushing interval summarizations")
			}
		} else {
			if err := impl.recorder.Flush(); err != nil {
				return errors.Wrap(err, "problem flushing recorder")
			}
		}

		if err := ftdc.FlushCollector(impl.collector, impl.file); err != nil {
			return errors.Wrap(err, "problem writing collector contents to file")
		}
		if err := impl.file.Close(); err != nil {
			return errors.Wrap(err, "problem closing open file")
		}
	}

	delete(r.cache, key)
	return nil
}

// CreateOptions support the use and creation of a collector.
type CreateOptions struct {
	Path      string
	ChunkSize int
	Streaming bool
	Dynamic   bool
	Recorder  RecorderType
}

func (opts *CreateOptions) build() (*recorderInstance, error) {
	if err := opts.Recorder.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid recorder type")
	}

	if opts.Recorder == CustomMetrics && !opts.Dynamic {
		return nil, errors.New("cannot use the custom metrics collector with a non-dynamic collector")
	}

	if _, err := os.Stat(opts.Path); !os.IsNotExist(err) {
		return nil, errors.Errorf("could not create '%s' because it exists", opts.Path)
	}

	file, err := os.Create(opts.Path)
	if err != nil {
		return nil, errors.Wrapf(err, "problem opening file '%s'", opts.Path)
	}

	out := &recorderInstance{
		isDynamic: opts.Dynamic,
		file:      file,
	}

	switch {
	case opts.Streaming && opts.Dynamic:
		out.collector = ftdc.NewStreamingDynamicCollector(opts.ChunkSize, file)
	case !opts.Streaming && opts.Dynamic:
		out.collector = ftdc.NewDynamicCollector(opts.ChunkSize)
	case opts.Streaming && !opts.Dynamic:
		out.collector = ftdc.NewStreamingCollector(opts.ChunkSize, file)
	case !opts.Streaming && !opts.Dynamic:
		out.collector = ftdc.NewBatchCollector(opts.ChunkSize)
	default:
		return nil, errors.New("invalid collector defined")
	}

	switch opts.Recorder {
	case RecorderPerf:
		out.recorder = events.NewRawRecorder(out.collector)
	case RecorderPerfSingle:
		out.recorder = events.NewSingleRecorder(out.collector)
	case RecorderPerf100ms:
		out.recorder = events.NewGroupedRecorder(out.collector, 100*time.Millisecond)
	case RecorderPerf1s:
		out.recorder = events.NewGroupedRecorder(out.collector, time.Second)
	case RecorderHistogramSingle:
		out.recorder = events.NewSingleHistogramRecorder(out.collector)
	case RecorderHistogram100ms:
		out.recorder = events.NewHistogramGroupedRecorder(out.collector, 100*time.Millisecond)
	case RecorderHistogram1s:
		out.recorder = events.NewHistogramGroupedRecorder(out.collector, time.Second)
	case CustomMetrics:
		out.isCustom = true
		out.tracker = &customEventTracker{Custom: events.MakeCustom(128)}
	default:
		return nil, errors.New("invalid recorder defined")
	}

	return out, nil
}
