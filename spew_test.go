package spew

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/transport/http/client"
)

var outputFolder = "./fixtures"

func TestTransport_nilResponse(t *testing.T) {
	expErr := errors.New("expect me")

	client := newClientFactory(nil, expErr)(context.Background())

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	res, err := client.Do(req)
	if res != nil {
		t.Errorf("unexpected response: %v", res)
	}
	if err == nil || err.Error() != "Get http://example.com: expect me" {
		t.Errorf("unexpected error. have: %v, want: %v", err, expErr)
	}
}

func TestTransport_nilResponseBody(t *testing.T) {
	expectedResponse := &http.Response{
		StatusCode: 200,
	}

	client := newClientFactory(expectedResponse, nil)(context.Background())

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
	expectedResponse := &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString("response body")),
	}

	client := newClientFactory(expectedResponse, nil)(context.Background())

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	res, err := client.Do(req)
	if res != expectedResponse {
		t.Errorf("unexpected response. have: %v, want: %v", res, expectedResponse)
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func newClientFactory(resp *http.Response, err error) client.HTTPClientFactory {
	logger, _ := logging.NewLogger("DEBUG", ioutil.Discard, "")
	cf := func(_ context.Context) *http.Client {
		return &http.Client{
			Transport: &mockedRoundTripper{
				resp: resp,
				err:  err,
			},
		}
	}
	return ClientFactory(logger, cf, outputFolder)
}

type mockedRoundTripper struct {
	err  error
	resp *http.Response
}

func (m *mockedRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return m.resp, m.err
}
