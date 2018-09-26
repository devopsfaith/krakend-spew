// Package spew provides a set of middlewares for the KrakenD framework ready to start dumping
// (pretty printed) request and response pairs into files
//
package spew

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"path"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
)

// New returns a proxy middleware ready to start dumping all the requests and responses
// it processes.
//
// The txt files will be stored in the path defined by the output argument, using the name argument
// as a prefix, so a pair request and response named "xxx" will be stored in the output folder as xxx_{timestamp}.txt
func New(logger logging.Logger, output, name string) proxy.Middleware {
	d := Dumper{
		Path:   path.Join(output, name),
		Logger: logger,
	}

	return func(next ...proxy.Proxy) proxy.Proxy {
		switch len(next) {
		case 0:
			panic(proxy.ErrNotEnoughProxies)
		case 1:
		default:
			panic(proxy.ErrTooManyProxies)
		}
		return func(ctx context.Context, req *proxy.Request) (*proxy.Response, error) {
			resp, err := next[0](ctx, req)

			logger.Debug("spew: capturing request and response at the", name, "layer")
			d.Dump(req, resp, err)

			return resp, err
		}
	}
}

// ProxyFactory returns a proxy.FactoryFunc over the received proxy.FactoryFunc with a spew middleware wrapping
// the generated pipe
func ProxyFactory(logger logging.Logger, factory proxy.Factory, output string) proxy.FactoryFunc {
	return func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		p, err := factory.New(cfg)
		if err != nil {
			return p, err
		}

		name := "proxy_" + base64.URLEncoding.EncodeToString([]byte(cfg.Endpoint))
		return New(logger, output, name)(p), nil
	}
}

// BackendFactory returns a proxy.BackendFactory over the received proxy.BackendFactory with a spew middleware wrapping
// the generated backend
func BackendFactory(logger logging.Logger, factory proxy.BackendFactory, output string) proxy.BackendFactory {
	return func(backend *config.Backend) proxy.Proxy {
		name := "backend_" + base64.URLEncoding.EncodeToString([]byte(backend.URLPattern))
		return New(logger, output, name)(factory(backend))
	}
}

// ClientFactory decorates the transport of the client generated by the received factory with a Transport
func ClientFactory(l logging.Logger, f proxy.HTTPClientFactory, o string) proxy.HTTPClientFactory {
	return func(ctx context.Context) *http.Client {
		c := f(ctx)

		next := c.Transport

		if next == nil {
			next = http.DefaultTransport
		}

		c.Transport = &Transport{
			Transport: next,
			Logger:    l,
			Output:    o,
		}

		return c
	}
}

// Transport is a wrapper over an instance of http.RoundTripper. It dumps every pair of request
// and response into a dedicated file at the designed output folder
type Transport struct {
	Transport http.RoundTripper
	Logger    logging.Logger
	Output    string
}

// RoundTrip takes a Request and returns a Response
//
// It delegates the actual execution and it just dumps the request, the response and the possible
// error
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	resp, err = t.Transport.RoundTrip(req)

	t.Logger.Debug("spew: capturing http request and response at the client layer")
	name := base64.URLEncoding.EncodeToString([]byte(req.URL.String()))

	t.basicDump(name, req, resp, err)

	d := Dumper{
		Logger: t.Logger,
		Path:   path.Join(t.Output, "client_"+name),
	}

	d.Dump(req, resp, err)

	return
}

// basicDump dumps just what is in the wire
func (t *Transport) basicDump(name string, req *http.Request, resp *http.Response, err error) {
	in := new(bytes.Buffer)

	writeHeader(in, "Request")
	dump, _ := httputil.DumpRequestOut(req, true)
	in.Write(dump)

	writeHeader(in, "Response")
	dump2, _ := httputil.DumpResponse(resp, true)
	in.Write(dump2)

	writeHeader(in, "error")
	if err != nil {
		in.Write([]byte(err.Error()))
	}

	go writeToFile(in, path.Join(t.Output, "client_basic_"+name), t.Logger)
}

// RunServerFunc is the interface expected by all the KrakenD http routers
type RunServerFunc func(context.Context, config.ServiceConfig, http.Handler) error

// RunServer returns a wrapper over the received RunServerFunc so it can inject a decorated
// http.Handler and dump pairs of request and response into dedicated files
func RunServer(l logging.Logger, f RunServerFunc, o string) RunServerFunc {
	return func(ctx context.Context, cfg config.ServiceConfig, handler http.Handler) error {
		return f(ctx, cfg, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			handler.ServeHTTP(rw, req)

			d := Dumper{
				Path:   path.Join(o, "router_"+base64.URLEncoding.EncodeToString([]byte(req.URL.String()))),
				Logger: l,
			}

			l.Debug("spew: capturing http request and response at the router layer")
			d.Dump(req, rw, nil)
		}))
	}
}

// Dumper is the wrapper over the spew pretty printer
type Dumper struct {
	Path   string
	Logger logging.Logger
}

// Dump dumps a request-response-error tuple into a file
func (d *Dumper) Dump(req interface{}, resp interface{}, err error) {
	bf := new(bytes.Buffer)

	writeHeader(bf, "Request")
	spew.Fdump(bf, req)

	writeHeader(bf, "Response")
	spew.Fdump(bf, resp)

	writeHeader(bf, "error")
	spew.Fdump(bf, err)

	go writeToFile(bf, d.Path, d.Logger)
}

const lineSeparation = "\n*************************************************************\n"

func writeHeader(w io.Writer, msg string) {
	w.Write([]byte(lineSeparation + msg + lineSeparation))
}

func writeToFile(in *bytes.Buffer, preffix string, logger logging.Logger) {
	now := time.Now().UnixNano()
	if err := ioutil.WriteFile(fmt.Sprintf("%s_%d.txt", preffix, now), in.Bytes(), 0666); err != nil {
		logger.Error("spew: writing the captured data:", err.Error())
	}
}