package types

import (
	"encoding/gob"
	"io"
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
