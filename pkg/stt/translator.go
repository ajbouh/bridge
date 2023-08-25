package stt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

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
func (s *HttpTranslator) Translate(audio []*CapturedAudio, language string) (*Transcription, error) {
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
		transcription.AudioSources = audio
		transcription.StartTimestamp = audio[0].StartTimestamp
		transcription.EndTimestamp = audio[0].EndTimestamp
		err = json.Unmarshal(body, &transcription)
		return &transcription, err
	} else {
		return nil, fmt.Errorf("error: %s", body)
	}
}

func (s *HttpTranslator) Run(transcriptionStream chan<- *Transcription, listener <-chan Document) {
	for doc := range listener {
		t := doc.Transcriptions[len(doc.Transcriptions)-1]
		if t.Language == "en" || t.Language == "" || len(t.AudioSources) == 0 || t.TranscriptSources != nil {
			continue
		}

		fmt.Printf("Foreign language detected language=%s translating to English...\n", t.Language)

		transcript, err := s.Translate(t.AudioSources, "en")
		if err != nil {
			fmt.Printf("error translating: %s", err)
			continue
		}
		transcript.TranscriptSources = []*Transcription{t}
		transcript.AllLanguageProbs = nil

		for i := range transcript.Segments {
			transcript.Segments[i].Speaker = "Translator"
			transcript.Segments[i].IsAssistant = true
		}

		transcriptionStream <- transcript
	}
}
