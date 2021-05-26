package spew

import (
	"context"
	"errors"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/luraproject/lura/config"
	"github.com/luraproject/lura/logging"
	"github.com/luraproject/lura/proxy"
)

var outputFolder = "./fixtures"

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

func tearDown(f func()) {
	f()
	fileFlusher = flusher{in: make(chan dumpedItem, 100)}
	once = new(sync.Once)
}
