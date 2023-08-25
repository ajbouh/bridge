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

func (s *HttpTranscriber) Transcribe(audio []*CapturedAudio) (*Transcription, error) {
	payloadBytes, err := json.Marshal(audio[0].PCM)
	if err != nil {
		return nil, err
	}

	// Send POST request to the API
	resp, err := http.Post(s.url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check the response status code
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		transcription := Transcription{}
		err = json.Unmarshal(body, &transcription)
		transcription.AudioSources = audio
		transcription.StartTimestamp = audio[0].StartTimestamp
		transcription.EndTimestamp = audio[0].EndTimestamp
		return &transcription, err
	} else {
		return nil, fmt.Errorf("error: %s", body)
	}
}

func (s *HttpTranscriber) Run(transcriptionStream chan<- *Transcription, audioStream <-chan *CapturedAudio) {
	for audio := range audioStream {
		// we have not been speaking for at least 500ms now so lets run inference
		fmt.Printf("transcribing with %d window length\n", len(audio.PCM))

		audioSources := []*CapturedAudio{audio}
		transcript, err := s.Transcribe(audioSources)
		if err != nil {
			fmt.Printf("error transcribing: %s\n", err)
			continue
		}
		transcript.AllLanguageProbs = nil

		for i := range transcript.Segments {
			transcript.Segments[i].Speaker = "Unknown"
			transcript.Segments[i].IsAssistant = false
		}

		transcriptionStream <- transcript
	}
}
