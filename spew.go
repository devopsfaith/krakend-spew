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
	"path"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/luraproject/lura/config"
	"github.com/luraproject/lura/logging"
	"github.com/luraproject/lura/proxy"
)

// New returns a proxy middleware ready to start dumping all the requests and responses
// it processes.
func New(logger logging.Logger, name string, dumper Dumper) proxy.Middleware {
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
			dumper.Dump(name, req, resp, err)

			return resp, err
		}
	}
}

// ProxyFactory returns a proxy.FactoryFunc over the received proxy.FactoryFunc with a spew middleware wrapping
// the generated pipe
func ProxyFactory(logger logging.Logger, factory proxy.Factory, df DumperFactory) proxy.FactoryFunc {
	dumper := df(SpewFormater)

	return func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		p, err := factory.New(cfg)
		if err != nil {
			return p, err
		}

		name := "proxy_" + base64.URLEncoding.EncodeToString([]byte(cfg.Endpoint))
		mw := New(logger, name, dumper)
		return mw(p), nil
	}
}

// BackendFactory returns a proxy.BackendFactory over the received proxy.BackendFactory with a spew middleware wrapping
// the generated backend
func BackendFactory(logger logging.Logger, factory proxy.BackendFactory, df DumperFactory) proxy.BackendFactory {
	dumper := df(SpewFormater)

	return func(backend *config.Backend) proxy.Proxy {
		name := "backend_" + base64.URLEncoding.EncodeToString([]byte(backend.URLPattern))
		mw := New(logger, name, dumper)
		return mw(factory(backend))
	}
}

// Dumper is the interface for the structs responsibles of persisting the inspected pair req/resp after formating them
type Dumper interface {
	Dump(id string, req interface{}, resp interface{}, err error)
}

// DumperFactory is the signature of a function that returns a dumper with the received formater injected.
type DumperFactory func(formater Formater) Dumper

// NewFileDumperFactory creates a DumperFactory for building dumpers writing the intercepted data as txt into the filesystem.
//
// The txt files will be stored in the path defined by the output argument, using the name argument
// as a prefix, so a pair request and response named "xxx" will be stored in the output folder as xxx_{timestamp}.txt
func NewFileDumperFactory(ctx context.Context, path string, l logging.Logger) DumperFactory {
	once.Do(func() {
		fileFlusher = flusher{in: make(chan dumpedItem, 100)}
		for i := 0; i < fileWriterWorkers; i++ {
			go fileFlusher.consume(ctx, l)
		}
	})
	return func(formater Formater) Dumper {
		return &fileDumper{
			path:     path,
			l:        l,
			formater: formater,
			out:      fileFlusher.in,
		}
	}
}

type fileDumper struct {
	path     string
	l        logging.Logger
	formater Formater
	out      chan dumpedItem
}

func (f *fileDumper) Dump(name string, req interface{}, resp interface{}, err error) {
	d := dumpedItem{
		path:    fmt.Sprintf("%s_%d.txt", path.Join(f.path, name), time.Now().UnixNano()),
		content: f.formater(req, resp, err),
		l:       f.l,
	}
	select {
	case f.out <- d:
	default:
	}
}

// Formater is the signature of a function that transform the inspected data into a byte array representation
type Formater func(req interface{}, resp interface{}, err error) []byte

// SpewFormater is a function that dumps the inspected data using the spew lib
func SpewFormater(req interface{}, resp interface{}, err error) []byte {
	bf := new(bytes.Buffer)

	writeHeader(bf, "Request")
	spew.Fdump(bf, req)

	writeHeader(bf, "Response")
	spew.Fdump(bf, resp)

	writeHeader(bf, "error")
	spew.Fdump(bf, err)

	return bf.Bytes()
}

const lineSeparation = "\n*************************************************************\n"

func writeHeader(w io.Writer, msg string) {
	w.Write([]byte(lineSeparation + msg + lineSeparation))
}

type dumpedItem struct {
	path    string
	content []byte
	l       logging.Logger
}

var (
	fileFlusher       flusher
	once              = new(sync.Once)
	fileWriterWorkers = 5
)

type flusher struct {
	in chan dumpedItem
}

func (f flusher) consume(ctx context.Context, l logging.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case i := <-f.in:
			if err := ioutil.WriteFile(i.path, i.content, 0666); err != nil && i.l != nil {
				l.Error("spew: writing the captured data:", err.Error())
			}
		}
	}
}
