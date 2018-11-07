package jasper

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/evergreen-ci/gimlet"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/mongodb/grip/send"
	"github.com/pkg/errors"
	"github.com/tychoish/lru"
)

// Service defines a REST service that provides a remote manager, using
// gimlet to publish routes.
type Service struct {
	hostID     string
	manager    Manager
	cache      *lru.Cache
	cacheOpts  CacheOptions
	cacheMutex sync.RWMutex
}

// NewManagerService creates a service object around an existing
// manager. You must access the application and routes via the App()
// method separately. The constructor wraps basic managers with a
// manager implementation that does locking.
func NewManagerService(m Manager) *Service {
	if bpm, ok := m.(*basicProcessManager); ok {
		m = &localProcessManager{manager: bpm}
	}

	return &Service{
		manager: m,
	}
}

const (
	// DefaultCachePruneDelay is the duration between LRU cache prunes.
	DefaultCachePruneDelay = 10 * time.Second
	// DefaultMaxCacheSize is the maximum allowed size of the LRU cache.
	DefaultMaxCacheSize = 1024 * 1024 * 1024
)

// App constructs and returns a gimlet application for this
// service. It attaches no middleware and does not start the service.
func (s *Service) App() *gimlet.APIApp {
	s.hostID, _ = os.Hostname()
	s.cache = lru.NewCache()
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.cacheOpts.PruneDelay = DefaultCachePruneDelay
	s.cacheOpts.MaxSize = DefaultMaxCacheSize
	s.cacheOpts.Disabled = false

	app := gimlet.NewApp()

	app.AddRoute("/").Version(1).Get().Handler(s.rootRoute)
	app.AddRoute("/create").Version(1).Post().Handler(s.createProcess)
	app.AddRoute("/download").Version(1).Post().Handler(s.downloadFile)
	app.AddRoute("/download/cache").Version(1).Post().Handler(s.configureCache)
	app.AddRoute("/download/mongodb").Version(1).Post().Handler(s.downloadMongoDB)
	app.AddRoute("/list/{filter}").Version(1).Get().Handler(s.listProcesses)
	app.AddRoute("/list/group/{name}").Version(1).Get().Handler(s.listGroupMembers)
	app.AddRoute("/process/{id}").Version(1).Get().Handler(s.getProcess)
	app.AddRoute("/process/{id}/buildlogger-urls").Version(1).Get().Handler(s.getBuildloggerURLs)
	app.AddRoute("/process/{id}/tags").Version(1).Get().Handler(s.getProcessTags)
	app.AddRoute("/process/{id}/tags").Version(1).Delete().Handler(s.deleteProcessTags)
	app.AddRoute("/process/{id}/tags").Version(1).Post().Handler(s.addProcessTag)
	app.AddRoute("/process/{id}/wait").Version(1).Get().Handler(s.waitForProcess)
	app.AddRoute("/process/{id}/metrics").Version(1).Get().Handler(s.processMetrics)
	app.AddRoute("/process/{id}/signal/{signal}").Version(1).Patch().Handler(s.signalProcess)
	app.AddRoute("/process/{id}/logs").Version(1).Get().Handler(s.getLogs)
	app.AddRoute("/close").Version(1).Delete().Handler(s.closeManager)

	go s.backgroundPrune()

	return app
}

func (s *Service) backgroundPrune() {
	s.cacheMutex.RLock()
	timer := time.NewTimer(s.cacheOpts.PruneDelay)
	s.cacheMutex.RUnlock()

	for {
		<-timer.C
		s.cacheMutex.RLock()
		if !s.cacheOpts.Disabled {
			if err := s.cache.Prune(s.cacheOpts.MaxSize, nil, false); err != nil {
				grip.Error(errors.Wrap(err, "error during cache pruning"))
			}
		}
		timer.Reset(s.cacheOpts.PruneDelay)
		s.cacheMutex.RUnlock()
	}
}

func writeError(rw http.ResponseWriter, err gimlet.ErrorResponse) {
	gimlet.WriteJSONResponse(rw, err.StatusCode, err)
}

func (s *Service) rootRoute(rw http.ResponseWriter, r *http.Request) {
	gimlet.WriteJSON(rw, struct {
		HostID string `json:"host_id"`
		Active bool   `json:"active"`
	}{
		HostID: s.hostID,
		Active: true,
	})

}

func (s *Service) createProcess(rw http.ResponseWriter, r *http.Request) {
	opts := &CreateOptions{}
	if err := gimlet.GetJSON(r.Body, opts); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrap(err, "problem reading request").Error(),
		})
		return
	}

	if err := opts.Validate(); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrap(err, "invalid creation options").Error(),
		})
		return
	}

	var ctx context.Context
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), opts.Timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	proc, err := s.manager.Create(ctx, opts)
	if err != nil {
		cancel()
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrap(err, "problem submitting request").Error(),
		})
		return
	}

	if err := proc.RegisterTrigger(ctx, func(_ ProcessInfo) {
		cancel()
	}); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    errors.Wrap(err, "problem managing resources").Error(),
		})
		return
	}

	gimlet.WriteJSON(rw, proc.Info(r.Context()))
}

func (s *Service) getBuildloggerURLs(rw http.ResponseWriter, r *http.Request) {
	id := gimlet.GetVars(r)["id"]
	ctx := r.Context()

	proc, err := s.manager.Get(ctx, id)
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrapf(err, "no process '%s' found", id).Error(),
		})
		return
	}

	info := proc.Info(ctx)
	urls := []string{}
	for _, logger := range info.Options.Output.Loggers {
		if logger.Type == LogBuildloggerV2 || logger.Type == LogBuildloggerV3 {
			urls = append(urls, logger.Options.BuildloggerOptions.GetGlobalLogURL())
		}
	}

	if len(urls) == 0 {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Errorf("process '%s' does not use buildlogger", id).Error(),
		})
		return
	}

	gimlet.WriteJSON(rw, urls)
}

func (s *Service) listProcesses(rw http.ResponseWriter, r *http.Request) {
	filter := Filter(gimlet.GetVars(r)["filter"])
	if err := filter.Validate(); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrap(err, "invalid input").Error(),
		})
		return
	}

	ctx := r.Context()

	procs, err := s.manager.List(ctx, filter)
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Message:    err.Error(),
		})
		return
	}

	out := []ProcessInfo{}
	for _, proc := range procs {
		out = append(out, proc.Info(ctx))
	}

	gimlet.WriteJSON(rw, out)
}

func (s *Service) listGroupMembers(rw http.ResponseWriter, r *http.Request) {
	name := gimlet.GetVars(r)["name"]

	ctx := r.Context()

	procs, err := s.manager.Group(ctx, name)
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Message:    err.Error(),
		})
		return
	}

	out := []ProcessInfo{}
	for _, proc := range procs {
		out = append(out, proc.Info(ctx))
	}

	gimlet.WriteJSON(rw, out)
}

func (s *Service) getProcess(rw http.ResponseWriter, r *http.Request) {
	id := gimlet.GetVars(r)["id"]
	ctx := r.Context()
	proc, err := s.manager.Get(ctx, id)
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Message:    errors.Wrapf(err, "no process '%s' found", id).Error(),
		})
		return
	}

	info := proc.Info(ctx)
	if info.ID == "" {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Message:    fmt.Sprintf("no process '%s' found", id),
		})
		return
	}

	gimlet.WriteJSON(rw, info)
}

func (s *Service) processMetrics(rw http.ResponseWriter, r *http.Request) {
	id := gimlet.GetVars(r)["id"]
	ctx := r.Context()
	proc, err := s.manager.Get(ctx, id)
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Message:    errors.Wrapf(err, "no process '%s' found", id).Error(),
		})
		return
	}

	info := proc.Info(ctx)
	gimlet.WriteJSON(rw, message.CollectProcessInfoWithChildren(int32(info.PID)))
}

func (s *Service) getProcessTags(rw http.ResponseWriter, r *http.Request) {
	id := gimlet.GetVars(r)["id"]
	ctx := r.Context()
	proc, err := s.manager.Get(ctx, id)
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Message:    errors.Wrapf(err, "no process '%s' found", id).Error(),
		})
		return
	}

	gimlet.WriteJSON(rw, proc.GetTags())
}

func (s *Service) deleteProcessTags(rw http.ResponseWriter, r *http.Request) {
	id := gimlet.GetVars(r)["id"]
	ctx := r.Context()
	proc, err := s.manager.Get(ctx, id)
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Message:    errors.Wrapf(err, "no process '%s' found", id).Error(),
		})
		return
	}

	proc.ResetTags()
	gimlet.WriteJSON(rw, struct{}{})
}

func (s *Service) addProcessTag(rw http.ResponseWriter, r *http.Request) {
	id := gimlet.GetVars(r)["id"]
	ctx := r.Context()
	proc, err := s.manager.Get(ctx, id)
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Message:    errors.Wrapf(err, "no process '%s' found", id).Error(),
		})
		return
	}

	newtags := r.URL.Query()["add"]
	if len(newtags) == 0 {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    "no new tags specified",
		})
		return
	}

	for _, t := range newtags {
		proc.Tag(t)
	}

	gimlet.WriteJSON(rw, struct{}{})
}

func (s *Service) waitForProcess(rw http.ResponseWriter, r *http.Request) {
	id := gimlet.GetVars(r)["id"]
	ctx := r.Context()
	proc, err := s.manager.Get(ctx, id)
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Message:    errors.Wrapf(err, "no process '%s' found", id).Error(),
		})
		return
	}

	if err := proc.Wait(ctx); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    err.Error(),
		})
		return
	}

	gimlet.WriteJSON(rw, struct{}{})
}

func (s *Service) signalProcess(rw http.ResponseWriter, r *http.Request) {
	vars := gimlet.GetVars(r)
	id := vars["id"]
	sig, err := strconv.Atoi(vars["signal"])
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrapf(err, "problem finding signal '%s'", vars["signal"]).Error(),
		})
		return
	}

	ctx := r.Context()
	proc, err := s.manager.Get(ctx, id)
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Message:    errors.Wrapf(err, "no process '%s' found", id).Error(),
		})
		return
	}

	if err := proc.Signal(ctx, syscall.Signal(sig)); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    err.Error(),
		})
		return
	}

	gimlet.WriteJSON(rw, struct{}{})
}

func (s *Service) downloadFile(rw http.ResponseWriter, r *http.Request) {
	var info DownloadInfo
	if err := gimlet.GetJSON(r.Body, &info); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrap(err, "problem reading request").Error(),
		})
		return
	}

	if err := info.Validate(); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrap(err, "problem validating download info").Error(),
		})
		return
	}

	if err := info.Download(); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrapf(err, "problem occurred during file download for URL %s", info.URL).Error(),
		})
		return
	}

	gimlet.WriteJSON(rw, struct{}{})
}

func (s *Service) getLogs(rw http.ResponseWriter, r *http.Request) {
	vars := gimlet.GetVars(r)
	id := vars["id"]
	ctx := r.Context()

	proc, err := s.manager.Get(ctx, id)
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Message:    errors.Wrapf(err, "no process '%s' found", id).Error(),
		})
		return
	}

	info := proc.Info(ctx)
	// Implicitly assumes that there's at most 1 in-memory logger.
	var inMemorySender *send.InMemorySender
	for _, logger := range info.Options.Output.Loggers {
		sender, ok := logger.sender.(*send.InMemorySender)
		if ok {
			inMemorySender = sender
			break
		}
	}

	if inMemorySender == nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    errors.Errorf("no in-memory logger found for process '%s'", id).Error(),
		})
		return
	}

	logs, err := inMemorySender.GetString()
	if err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Message:    err.Error(),
		})
		return
	}

	gimlet.WriteJSON(rw, logs)
}

func (s *Service) closeManager(rw http.ResponseWriter, r *http.Request) {
	if err := s.manager.Close(r.Context()); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    err.Error(),
		})
		return
	}

	gimlet.WriteJSON(rw, struct{}{})
}

func (s *Service) downloadMongoDB(rw http.ResponseWriter, r *http.Request) {
	opts := MongoDBDownloadOptions{}
	if err := gimlet.GetJSON(r.Body, &opts); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrap(err, "problem reading request").Error(),
		})
		return
	}

	if err := opts.Validate(); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrap(err, "problem validating MongoDB download options").Error(),
		})
		return
	}

	if err := SetupDownloadMongoDBReleases(r.Context(), s.cache, opts); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    errors.Wrap(err, "problem in download setup").Error(),
		})
		return
	}

	gimlet.WriteJSON(rw, struct{}{})
}

func (s *Service) configureCache(rw http.ResponseWriter, r *http.Request) {
	opts := CacheOptions{}
	if err := gimlet.GetJSON(r.Body, &opts); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrap(err, "problem reading request").Error(),
		})
		return
	}

	if err := opts.Validate(); err != nil {
		writeError(rw, gimlet.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    errors.Wrap(err, "problem validating cache options").Error(),
		})
		return
	}

	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	if opts.MaxSize > 0 {
		s.cacheOpts.MaxSize = opts.MaxSize
	}
	if opts.PruneDelay > time.Duration(0) {
		s.cacheOpts.PruneDelay = opts.PruneDelay
	}
	s.cacheOpts.Disabled = opts.Disabled

	gimlet.WriteJSON(rw, struct{}{})
}
