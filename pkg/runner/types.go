package runner

import (
	"encoding/gob"
	"io"

	v8 "rogchap.com/v8go"
)

type RtnValType string

const (
	RtnValueTypeNil  RtnValType = "nil"
	RtnValueTypeJSON RtnValType = "json"
)

type RunCodeRequest struct {
	ID           string     `json:"id"`
	Code         string     `json:"code"`
	ResponseType RtnValType `json:"responseType"`
}

type RunCodeResponse struct {
	ID     string  `json:"id"`
	Error  *string `json:"error,omitempty"`
	Result *string `json:"result,omitempty"`
}

func errResult(id string, err error) RunCodeResponse {
	errStr := err.Error()
	return RunCodeResponse{
		ID:    id,
		Error: &errStr,
	}
}

func nilResult(id string) RunCodeResponse {
	return RunCodeResponse{
		ID: id,
	}
}

func jsonResult(id string, codeCtx *v8.Context, val *v8.Value) RunCodeResponse {
	jsonStr, err := v8.JSONStringify(codeCtx, val)
	if err != nil {
		return errResult(id, err)
	}
	return RunCodeResponse{
		ID:     id,
		Result: &jsonStr,
	}
}

// NewRunCodeRequestEncoder creates a new encoder for RunCodeRequest.
// NOTE: only one encoder should be created for a writer.
func NewRunCodeRequestEncoder(w io.Writer) *gob.Encoder {
	return gob.NewEncoder(w)
}

// NewReadRunCodeResponseDecoder creates a new decoder for RunCodeResponse.
// NOTE: only one decoder should be created for a reader.
func NewReadRunCodeResponseDecoder(r io.Reader) *gob.Decoder {
	return gob.NewDecoder(r)
}
