package handler

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/grepplabs/backstream/internal/message"
	"google.golang.org/protobuf/types/known/structpb"
)

func fromHttpResponse(resp *http.Response) (*message.EventHTTPResponse, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	headers, err := toHeaders(resp.Header)
	if err != nil {
		return nil, err
	}
	return &message.EventHTTPResponse{
		StatusCode: int32(resp.StatusCode),
		Body:       body,
		Headers:    headers,
	}, nil
}

func fromHttpRequest(req *http.Request) (*message.EventHTTPRequest, error) {
	defer req.Body.Close()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	rawPath := req.URL.RawPath
	if rawPath == "" {
		rawPath = req.URL.Path
	}
	headers, err := toHeaders(req.Header)
	if err != nil {
		return nil, err
	}
	return &message.EventHTTPRequest{
		Method:   req.Method,
		RawPath:  rawPath,
		RawQuery: req.URL.RawQuery,
		Headers:  headers,
		Body:     body,
	}, nil
}

func toHttpRequest(event *message.EventHTTPRequest) (*http.Request, error) {
	u := url.URL{
		Scheme:   "http",
		Host:     "localhost",
		RawQuery: event.RawQuery,
		Path:     event.RawPath,
	}
	request, err := http.NewRequest(event.Method, u.String(), bytes.NewReader(event.Body))
	if err != nil {
		return nil, err
	}
	request.Header, err = fromHeaders(event.Headers)
	if err != nil {
		return nil, err
	}
	return request, nil
}

func writeHttpResponse(w http.ResponseWriter, event *message.EventHTTPResponse) error {
	header, err := fromHeaders(event.Headers)
	if err != nil {
		return err
	}
	for key, values := range header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(int(event.StatusCode))
	_, err = io.Copy(w, bytes.NewReader(event.Body))
	return err
}

func toHttpResponse(event *message.EventHTTPResponse) (*http.Response, error) {
	header, err := fromHeaders(event.Headers)
	if err != nil {
		return nil, err
	}
	response := &http.Response{
		StatusCode: int(event.StatusCode),
		Header:     header,
		Body:       io.NopCloser(bytes.NewReader(event.Body)),
	}
	return response, nil
}

func toHeaders(hs http.Header) (map[string]*structpb.ListValue, error) {
	headers := make(map[string]*structpb.ListValue)
	var err error
	for k, vs := range hs {
		headers[k], err = toListString(vs)
		if err != nil {
			return nil, err
		}
	}
	return headers, nil
}

func fromHeaders(hs map[string]*structpb.ListValue) (http.Header, error) {
	header := make(http.Header)
	var err error
	for k, vs := range hs {
		header[k], err = fromListString(vs)
		if err != nil {
			return nil, err
		}
	}
	return header, nil
}

func toListString(vs []string) (*structpb.ListValue, error) {
	var is = make([]interface{}, len(vs))
	for i, s := range vs {
		is[i] = s
	}
	return structpb.NewList(is)
}

func fromListString(vs *structpb.ListValue) (result []string, err error) {
	if vs == nil {
		return
	}
	for _, v := range vs.AsSlice() {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("header value not a string")
		}
		result = append(result, s)
	}
	return result, nil
}
