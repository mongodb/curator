package jasper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"syscall"

	"github.com/evergreen-ci/gimlet"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
)

type restClient struct {
	prefix string
	client *http.Client
}

func (c *restClient) getURL(route string, args ...interface{}) string {
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}

	if len(args) == 0 {
		return c.prefix + route
	}

	return fmt.Sprintf(c.prefix+route, args...)
}

func makeBody(data interface{}) (io.Reader, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "problem marshaling request body")
	}

	return bytes.NewBuffer(payload), nil
}

func handleError(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	gimerr := gimlet.ErrorResponse{}
	if err := gimlet.GetJSON(resp.Body, &gimerr); err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(gimerr)
}

func (c *restClient) doRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "problem building request")
	}

	req = req.WithContext(ctx)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "problem making request")
	}
	if err = handleError(resp); err != nil {
		defer resp.Body.Close()
		return nil, errors.WithStack(err)
	}

	return resp, nil
}

func (c *restClient) Create(ctx context.Context, opts *CreateOptions) (Process, error) {
	body, err := makeBody(opts)
	if err != nil {
		return nil, errors.Wrap(err, "problem building request for job create")
	}

	req, err := http.NewRequest(http.MethodPost, c.getURL("/create"), body)
	if err != nil {
		return nil, errors.Wrap(err, "problem building request")
	}
	req = req.WithContext(ctx)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "problem making request")
	}

	if err = handleError(resp); err != nil {
		return nil, errors.WithStack(err)
	}

	var info ProcessInfo
	if err := gimlet.GetJSON(resp.Body, &info); err != nil {
		return nil, errors.Wrap(err, "problem reading process info from response")
	}

	return &restProcess{
		id:     info.ID,
		client: c,
	}, nil
}

func (c *restClient) Register(ctx context.Context, proc Process) error {
	return errors.New("cannot register a local process on a remote service")
}

func (c *restClient) getListOfProcesses(req *http.Request) ([]Process, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "problem making request")
	}

	if err = handleError(resp); err != nil {
		return nil, errors.WithStack(err)
	}

	payload := []ProcessInfo{}
	if err := gimlet.GetJSON(resp.Body, &payload); err != nil {
		return nil, errors.Wrap(err, "problem reading process info from response")
	}

	output := []Process{}
	for _, info := range payload {

		output = append(output, &restProcess{
			id:     info.ID,
			client: c,
		})
	}

	return output, nil
}

func (c *restClient) List(ctx context.Context, f Filter) ([]Process, error) {
	if err := f.Validate(); err != nil {
		return nil, errors.WithStack(err)
	}

	req, err := http.NewRequest(http.MethodGet, c.getURL("/list/%s", string(f)), nil)
	if err != nil {
		return nil, errors.Wrap(err, "problem building request")
	}
	req = req.WithContext(ctx)

	out, err := c.getListOfProcesses(req)

	return out, errors.WithStack(err)
}

func (c *restClient) Group(ctx context.Context, name string) ([]Process, error) {
	req, err := http.NewRequest(http.MethodGet, c.getURL("/list/group/%s", name), nil)
	if err != nil {
		return nil, errors.Wrap(err, "problem building request")
	}

	req = req.WithContext(ctx)

	out, err := c.getListOfProcesses(req)

	return out, errors.WithStack(err)
}

func (c *restClient) getProcess(ctx context.Context, id string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.getURL("/process/%s", id), nil)
	if err != nil {
		return nil, errors.Wrap(err, "problem building request")
	}

	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "problem making request")
	}

	if err = handleError(resp); err != nil {
		return nil, errors.WithStack(err)
	}

	return resp, nil
}

func (c *restClient) getProcessInfo(ctx context.Context, id string) (ProcessInfo, error) {
	resp, err := c.getProcess(ctx, id)
	if err != nil {
		return ProcessInfo{}, errors.WithStack(err)
	}

	out := ProcessInfo{}
	if err = gimlet.GetJSON(resp.Body, &out); err != nil {
		return ProcessInfo{}, errors.WithStack(err)
	}

	return out, nil
}

func (c *restClient) Get(ctx context.Context, id string) (Process, error) {
	_, err := c.getProcess(ctx, id)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// we don't actually need to parse the body of the post if we
	// know the process exists.
	return &restProcess{
		id:     id,
		client: c,
	}, nil
}

func (c *restClient) Close(ctx context.Context) error {
	req, err := http.NewRequest(http.MethodDelete, c.getURL("/close"), nil)
	if err != nil {
		return errors.Wrap(err, "problem building request")
	}
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "problem making request")
	}

	if err = handleError(resp); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (c *restClient) GetBuildloggerURLs(ctx context.Context, id string) ([]string, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, c.getURL("/process/%s/buildlogger-urls", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	urls := []string{}
	if err = gimlet.GetJSON(resp.Body, &urls); err != nil {
		return nil, errors.Wrap(err, "problem reading urls from response")
	}

	return urls, nil
}

func (c *restClient) GetLogs(ctx context.Context, id string) ([]string, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, c.getURL("/process/%s/logs", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	logs := []string{}
	if err = gimlet.GetJSON(resp.Body, &logs); err != nil {
		return nil, errors.Wrap(err, "problem reading logs from response")
	}

	return logs, nil
}

func (c *restClient) DownloadFile(ctx context.Context, info DownloadInfo) error {
	body, err := makeBody(info)
	if err != nil {
		return errors.Wrap(err, "problem building request")
	}

	resp, err := c.doRequest(ctx, http.MethodPost, c.getURL("/download"), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *restClient) DownloadFileAsync(ctx context.Context, info DownloadInfo) error {
	body, err := makeBody(info)
	if err != nil {
		return errors.Wrap(err, "problem building request")
	}

	resp, err := c.doRequest(ctx, http.MethodPost, c.getURL("/download-async"), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// DownloadMongoDB downloads the desired version of MongoDB.
func (c *restClient) DownloadMongoDB(ctx context.Context, opts MongoDBDownloadOptions) error {
	body, err := makeBody(opts)
	if err != nil {
		return err
	}

	resp, err := c.doRequest(ctx, http.MethodPost, c.getURL("/download-mongodb"), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ConfigureCache changes the cache configurations.
func (c *restClient) ConfigureCache(ctx context.Context, opts CacheOptions) error {
	body, err := makeBody(opts)
	if err != nil {
		return err
	}

	resp, err := c.doRequest(ctx, http.MethodPost, c.getURL("/configure-cache"), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

type restProcess struct {
	id              string
	client          *restClient
	buildloggerURLs []string
}

func (p *restProcess) ID() string { return p.id }

func (p *restProcess) Info(ctx context.Context) ProcessInfo {
	info, err := p.client.getProcessInfo(ctx, p.id)
	grip.Debug(message.WrapError(err, message.Fields{"process": p.id}))
	return info
}

func (p *restProcess) Running(ctx context.Context) bool {
	info, err := p.client.getProcessInfo(ctx, p.id)
	grip.Debug(message.WrapError(err, message.Fields{"process": p.id}))
	return info.IsRunning
}

func (p *restProcess) Complete(ctx context.Context) bool {
	info, err := p.client.getProcessInfo(ctx, p.id)
	grip.Debug(message.WrapError(err, message.Fields{"process": p.id}))
	return info.Complete
}

func (p *restProcess) Signal(ctx context.Context, sig syscall.Signal) error {
	req, err := http.NewRequest(http.MethodPatch, p.client.getURL("/process/%s/signal/%d", p.id, sig), nil)
	if err != nil {
		return errors.Wrap(err, "problem building request")
	}

	req = req.WithContext(ctx)

	resp, err := p.client.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "problem making request")
	}

	if err = handleError(resp); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (p *restProcess) Wait(ctx context.Context) error {
	req, err := http.NewRequest(http.MethodGet, p.client.getURL("/process/%s/wait", p.id), nil)
	if err != nil {
		return errors.Wrap(err, "problem building request")
	}

	req = req.WithContext(ctx)

	resp, err := p.client.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "problem making request")
	}

	if err = handleError(resp); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (p *restProcess) RegisterTrigger(ctx context.Context, _ ProcessTrigger) error {
	return errors.New("cannot register triggers on remote processes")
}
func (p *restProcess) Tag(t string) {
	req, err := http.NewRequest(http.MethodPost, p.client.getURL("/process/%s/tags?add=%s", p.id, t), nil)
	if err != nil {
		grip.Debug(message.WrapError(err, message.Fields{
			"message": "problem making request",
			"process": p.id,
		}))
		return
	}

	resp, err := p.client.client.Do(req)
	if err != nil {
		grip.Debug(message.WrapError(err, message.Fields{
			"message": "problem making request",
			"process": p.id,
		}))
		return
	}

	if err = handleError(resp); err != nil {
		grip.Debug(message.WrapError(err, message.Fields{
			"message": "request returned error",
			"process": p.id,
		}))
		return
	}

	return
}

func (p *restProcess) GetTags() []string {
	req, err := http.NewRequest(http.MethodGet, p.client.getURL("/process/%s/tags", p.id), nil)
	if err != nil {
		grip.Debug(message.WrapError(err, message.Fields{
			"message": "problem making request",
			"process": p.id,
		}))
		return nil
	}

	resp, err := p.client.client.Do(req)
	if err != nil {
		grip.Debug(message.WrapError(err, message.Fields{
			"message": "problem making request",
			"process": p.id,
		}))
		return nil
	}

	if err = handleError(resp); err != nil {
		grip.Debug(message.WrapError(err, message.Fields{
			"message": "request returned error",
			"process": p.id,
		}))
		return nil
	}

	out := []string{}
	if err = gimlet.GetJSON(resp.Body, &out); err != nil {
		grip.Debug(message.WrapError(err, message.Fields{
			"message": "problem reading tags from response",
			"process": p.id,
		}))

		return nil
	}
	return out
}

func (p *restProcess) ResetTags() {
	req, err := http.NewRequest(http.MethodDelete, p.client.getURL("/process/%s/tags", p.id), nil)
	if err != nil {
		grip.Debug(message.WrapError(err, message.Fields{
			"message": "problem making request",
			"process": p.id,
		}))
		return
	}

	resp, err := p.client.client.Do(req)
	if err != nil {
		grip.Debug(message.WrapError(err, message.Fields{
			"message": "problem making request",
			"process": p.id,
		}))
		return
	}

	if err = handleError(resp); err != nil {
		grip.Debug(message.WrapError(err, message.Fields{
			"message": "request returned error",
			"process": p.id,
		}))
		return
	}
}
