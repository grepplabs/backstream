package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grepplabs/backstream/internal/message"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestFromHttpResponse(t *testing.T) {
	tests := []struct {
		name  string
		resp  *http.Response
		event *message.EventHTTPResponse
	}{
		{
			name: "http response",
			resp: &http.Response{
				StatusCode: 200,
				Header:     nil,
				Body:       io.NopCloser(strings.NewReader("")),
			},
			event: &message.EventHTTPResponse{
				StatusCode: 200,
				Headers:    map[string]*structpb.ListValue{},
				Body:       []byte(""),
			},
		},
		{
			name: "http response with headers",
			resp: &http.Response{
				StatusCode: 403,
				Header: map[string][]string{
					"key1": {"value1"},
				},
				Body: io.NopCloser(strings.NewReader("OK")),
			},
			event: &message.EventHTTPResponse{
				StatusCode: 403,
				Headers: map[string]*structpb.ListValue{
					"key1": mustToListString("value1"),
				},
				Body: []byte("OK"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event, err := fromHttpResponse(tc.resp)
			require.NoError(t, err)
			require.Equal(t, tc.event, event)
		})
	}
}

func TestFromHttpRequest(t *testing.T) {
	tests := []struct {
		name  string
		req   *http.Request
		event *message.EventHTTPRequest
	}{
		{
			name: "http request",
			req:  mustNewRequest("GET", "http://bing.com/search?q=dotnet", "", make(http.Header)),
			event: &message.EventHTTPRequest{
				Method:   "GET",
				RawPath:  "/search",
				RawQuery: "q=dotnet",
				Headers:  map[string]*structpb.ListValue{},
				Body:     []byte(""),
			},
		},
		{
			name: "http response with headers",
			req: mustNewRequest("POST", "http://bing.com", "OK", map[string][]string{
				"key1": {"value1"},
			}),
			event: &message.EventHTTPRequest{
				Method:   "POST",
				RawPath:  "",
				RawQuery: "",
				Headers: map[string]*structpb.ListValue{
					"key1": mustToListString("value1"),
				},
				Body: []byte("OK"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event, err := fromHttpRequest(tc.req)
			require.NoError(t, err)
			require.Equal(t, tc.event, event)
		})
	}
}

func TestToHttpRequest(t *testing.T) {
	tests := []struct {
		name  string
		event *message.EventHTTPRequest
		req   *http.Request
	}{
		{
			name: "http request",
			event: &message.EventHTTPRequest{
				Method:   "GET",
				RawPath:  "/search",
				RawQuery: "q=dotnet",
				Headers:  map[string]*structpb.ListValue{},
				Body:     []byte(""),
			},
			req: mustNewRequest("GET", "http://bing.com/search?q=dotnet", "", make(http.Header)),
		},
		{
			name: "http response with headers",
			req: mustNewRequest("POST", "http://bing.com", "OK", map[string][]string{
				"key1": {"value1"},
			}),
			event: &message.EventHTTPRequest{
				Method:   "POST",
				RawPath:  "",
				RawQuery: "",
				Headers: map[string]*structpb.ListValue{
					"key1": mustToListString("value1"),
				},
				Body: []byte("OK"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := toHttpRequest(tc.event)
			require.NoError(t, err)

			require.Equal(t, tc.req.Method, req.Method)
			require.Equal(t, tc.req.Header, req.Header)
			require.Equal(t, tc.req.URL.RawPath, req.URL.RawPath)
			require.Equal(t, tc.req.URL.Path, req.URL.Path)
			require.Equal(t, tc.req.URL.RawQuery, req.URL.RawQuery)

			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Equal(t, tc.event.Body, body)
		})
	}
}

func TestToHttpResponse(t *testing.T) {
	tests := []struct {
		name  string
		event *message.EventHTTPResponse
		resp  *http.Response
	}{
		{
			name: "http response",
			resp: &http.Response{
				StatusCode: 200,
				Header:     nil,
				Body:       io.NopCloser(strings.NewReader("")),
			},
			event: &message.EventHTTPResponse{
				StatusCode: 200,
				Headers:    map[string]*structpb.ListValue{},
				Body:       []byte(""),
			},
		},
		{
			name: "http response with headers",
			resp: &http.Response{
				StatusCode: 403,
				Header: map[string][]string{
					"key1": {"value1"},
				},
				Body: io.NopCloser(strings.NewReader("OK")),
			},
			event: &message.EventHTTPResponse{
				StatusCode: 403,
				Headers: map[string]*structpb.ListValue{
					"key1": mustToListString("value1"),
				},
				Body: []byte("OK"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := toHttpResponse(tc.event)
			require.NoError(t, err)
			require.Equal(t, int(tc.event.StatusCode), resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, tc.event.Body, body)

			header, err := fromHeaders(tc.event.Headers)
			require.NoError(t, err)
			require.Equal(t, header, resp.Header)

		})
	}
}

func TestWriteHttpResponse(t *testing.T) {
	tests := []struct {
		name  string
		event *message.EventHTTPResponse
	}{
		{
			name: "http response",
			event: &message.EventHTTPResponse{
				StatusCode: 200,
				Headers:    map[string]*structpb.ListValue{},
				Body:       []byte(""),
			},
		},
		{
			name: "http response with headers",
			event: &message.EventHTTPResponse{
				StatusCode: 403,
				Headers: map[string]*structpb.ListValue{
					"Key1": mustToListString("value1"),
				},
				Body: []byte("OK"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			rec := httptest.NewRecorder()
			err := writeHttpResponse(rec, tc.event)
			require.NoError(t, err)

			result := rec.Result()

			require.Equal(t, int(tc.event.StatusCode), result.StatusCode)

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			require.Equal(t, tc.event.Body, body)

			header, err := fromHeaders(tc.event.Headers)
			require.NoError(t, err)
			require.Equal(t, result.Header, header)
		})
	}
}

func TestHeaderConv(t *testing.T) {
	header := make(http.Header)
	header["Key1"] = []string{}
	header["Key2"] = []string{"value2_1"}
	header["Key3"] = []string{"value3_1", "value3_2"}

	hs, err := toHeaders(header)
	require.NoError(t, err)
	result, err := fromHeaders(hs)
	require.NoError(t, err)

	require.Len(t, result, 3)
	require.Len(t, result["Key1"], 0)
	require.Equal(t, result["Key2"], []string{"value2_1"})
	require.Equal(t, result["Key3"], []string{"value3_1", "value3_2"})
}

func mustToListString(vs ...string) *structpb.ListValue {
	l, err := toListString(vs)
	if err != nil {
		panic(err)
	}
	return l
}

func mustNewRequest(method, url string, body string, header http.Header) *http.Request {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header = header
	return req
}
