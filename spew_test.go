package spew

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	"github.com/devopsfaith/krakend/transport/http/client"
)

var outputFolder = "./fixtures"

func TestTransport_nilResponse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	defer tearDown(cancel)

	expErr := errors.New("expect me")

	client := newClientFactory(ctx, nil, expErr)(context.Background())

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	res, err := client.Do(req)
	if res != nil {
		t.Errorf("unexpected response: %v", res)
	}
	if err == nil || err.Error() != "Get http://example.com: expect me" {
		t.Errorf("unexpected error. have: %v, want: %v", err, expErr)
	}
}

func tearDown(f func()) {
	f()
	fileFlusher = flusher{in: make(chan dumpedItem, 100)}
	once = new(sync.Once)
}

func TestTransport_nilResponseBody(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	defer tearDown(cancel)

	expectedResponse := &http.Response{
		StatusCode: 200,
	}

	client := newClientFactory(ctx, expectedResponse, nil)(context.Background())

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	res, err := client.Do(req)
	if res != expectedResponse {
		t.Errorf("unexpected response. have: %v, want: %v", res, expectedResponse)
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTransport_nilError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	defer tearDown(cancel)

	expectedResponse := &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString("response body")),
	}

	client := newClientFactory(ctx, expectedResponse, nil)(context.Background())

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	res, err := client.Do(req)
	if res != expectedResponse {
		t.Errorf("unexpected response. have: %v, want: %v", res, expectedResponse)
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBackendFactory(t *testing.T) {
	logger, _ := logging.NewLogger("DEBUG", ioutil.Discard, "")
	ctx, cancel := context.WithCancel(context.Background())

	defer tearDown(cancel)

	for _, tc := range []struct {
		request  *proxy.Request
		response *proxy.Response
		err      string
	}{
		{
			request:  &proxy.Request{},
			response: &proxy.Response{},
		},
		{
			err: "some error",
		},
	} {
		bf := func(_ *config.Backend) proxy.Proxy {
			return func(_ context.Context, req *proxy.Request) (*proxy.Response, error) {
				if req != tc.request {
					t.Errorf("unexpected request: %v", req)
				}
				return tc.response, errors.New(tc.err)
			}
		}

		sbf := BackendFactory(logger, bf, NewFileDumperFactory(ctx, outputFolder, logger))
		resp, err := sbf(&config.Backend{URLPattern: "/a"})(context.Background(), tc.request)
		if err != nil && err.Error() != tc.err {
			t.Errorf("unexpected error: %v.", err)
		}
		if err == nil && tc.err != "" {
			t.Errorf("unexpected error: %v.", err)
		}
		if resp != tc.response {
			t.Errorf("unexpected response: %+v", *resp)
		}
	}
}

func TestProxyFactory(t *testing.T) {
	logger, _ := logging.NewLogger("DEBUG", ioutil.Discard, "")
	ctx, cancel := context.WithCancel(context.Background())

	defer tearDown(cancel)

	for _, tc := range []struct {
		request  *proxy.Request
		response *proxy.Response
		err      string
	}{
		{
			request:  &proxy.Request{},
			response: &proxy.Response{},
		},
		{
			err: "some error",
		},
	} {
		pf := proxy.FactoryFunc(func(_ *config.EndpointConfig) (proxy.Proxy, error) {
			return func(_ context.Context, req *proxy.Request) (*proxy.Response, error) {
				if req != tc.request {
					t.Errorf("unexpected request: %v", req)
				}
				return tc.response, errors.New(tc.err)
			}, nil
		})

		spf := ProxyFactory(logger, pf, NewFileDumperFactory(ctx, outputFolder, logger))
		prxy, err := spf(&config.EndpointConfig{Endpoint: "/a"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		resp, err := prxy(ctx, tc.request)
		if err != nil && err.Error() != tc.err {
			t.Errorf("unexpected error: %v.", err)
		}
		if err == nil && tc.err != "" {
			t.Errorf("unexpected error: %v.", err)
		}
		if resp != tc.response {
			t.Errorf("unexpected response: %+v", *resp)
		}
	}
}

func TestRunServer(t *testing.T) {
	logger, _ := logging.NewLogger("DEBUG", ioutil.Discard, "")
	ctx, cancel := context.WithCancel(context.Background())

	defer tearDown(cancel)

	cfg := config.ServiceConfig{Version: 1234}
	expectedHandler := new(dummyHandler)
	expectedErr := errors.New("expected error")

	mockedRunServerFunc := func(_ context.Context, cfg config.ServiceConfig, handler http.Handler) error {
		if cfg.Version != 1234 {
			return errors.New("unexpected config")
		}
		return expectedErr
	}
	rs := RunServer(logger, mockedRunServerFunc, NewFileDumperFactory(ctx, outputFolder, logger))

	if err := rs(ctx, cfg, expectedHandler); err != expectedErr {
		t.Errorf("unexpected error: %v", err)
	}
}

type dummyHandler struct{}

func (d *dummyHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(987)
}

func newClientFactory(ctx context.Context, resp *http.Response, err error) client.HTTPClientFactory {
	logger, _ := logging.NewLogger("DEBUG", ioutil.Discard, "")
	cf := func(_ context.Context) *http.Client {
		return &http.Client{
			Transport: &mockedRoundTripper{
				resp: resp,
				err:  err,
			},
		}
	}
	return ClientFactory(logger, cf, NewFileDumperFactory(ctx, outputFolder, logger))
}

type mockedRoundTripper struct {
	err  error
	resp *http.Response
}

func (m *mockedRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return m.resp, m.err
}
