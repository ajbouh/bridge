package stt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// ensure this satisfies the interface
var _ Transcriber = (*STTHttpBackend)(nil)

type STTHttpBackend struct {
	url string
}

func NewHTTPBackend(url string) (*STTHttpBackend, error) {
	if url == "" {
		return nil, fmt.Errorf("invalid url for STTHttpBackend %s", url)
	}
	return &STTHttpBackend{
		url: url,
	}, nil
}

func (s *STTHttpBackend) Transcribe(audioData []float32) (Transcription, error) {
	payloadBytes, err := json.Marshal(audioData)
	if err != nil {
		return Transcription{}, err
	}

	// Send POST request to the API
	resp, err := http.Post(s.url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return Transcription{}, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Transcription{}, err
	}

	// Check the response status code
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		transcription := Transcription{}
		err = json.Unmarshal(body, &transcription)
		return transcription, err
	} else {
		return Transcription{}, fmt.Errorf("error: %s", body)
	}
}
