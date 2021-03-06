package command

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	httptransport "github.com/go-kit/kit/transport/http"
)

type HTTPHandlers struct {
	NewCommandHandler http.Handler
}

func MakeHTTPHandlers(ctx context.Context, endpoints Endpoints, opts ...httptransport.ServerOption) HTTPHandlers {
	h := HTTPHandlers{
		NewCommandHandler: httptransport.NewServer(
			endpoints.NewCommandEndpoint,
			decodeRequest,
			encodeResponse,
			opts...,
		),
	}
	return h
}

type errorer interface {
	error() error
}

type statuser interface {
	status() int
}

// EncodeError is used by the HTTP transport to encode service errors in HTTP.
// The EncodeError should be passed to the Go-Kit httptransport as the
// ServerErrorEncoder to encode error responses with JSON.
func EncodeError(ctx context.Context, err error, w http.ResponseWriter) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	enc.Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func codeFromErr(err error) int {
	switch err {
	case errEmptyRequest:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func errorDecoder(r *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

type errorWrapper struct {
	Error string `json:"error"`
}

func decodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req newCommandRequest
	err := json.NewDecoder(io.LimitReader(r.Body, 10000)).Decode(&req)
	return req, err
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {

	if e, ok := response.(errorer); ok && e.error() != nil {
		EncodeError(ctx, e.error(), w)
		return nil
	}

	if s, ok := response.(statuser); ok {
		w.WriteHeader(s.status())
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(response)
}
