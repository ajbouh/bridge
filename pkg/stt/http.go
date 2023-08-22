package stt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// ensure this satisfies the interface
var _ Transcriber = (*HttpTranscriber)(nil)

type HttpTranscriber struct {
	url string
}

func NewHTTPTranscriber(url string) (*HttpTranscriber, error) {
	if url == "" {
		return nil, fmt.Errorf("invalid url for HttpTranscriber %s", url)
	}
	return &HttpTranscriber{
		url: url,
	}, nil
}

func (s *HttpTranscriber) Transcribe(audioData []float32) (Transcription, error) {
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

// ensure this satisfies the interface
var _ Translator = (*HttpTranslator)(nil)

type HttpTranslator struct {
	url string
}

func NewHTTPTranslator(url string) (*HttpTranslator, error) {
	if url == "" {
		return nil, fmt.Errorf("invalid url for HttpTranslator %s", url)
	}
	return &HttpTranslator{
		url: url,
	}, nil
}

// HACK For now we just assume it should be english! This is because that's all whisper can handle.
func (s *HttpTranslator) Translate(audioData []float32, language string) (Transcription, error) {
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
