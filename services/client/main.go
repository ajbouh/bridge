package main

import (
	"flag"
	"net/url"
	"os"

	"github.com/ajbouh/bridge/pkg/client"
	logr "github.com/ajbouh/bridge/pkg/log"

	stt "github.com/ajbouh/bridge/pkg/stt"

	"golang.org/x/exp/slog"
)

var (
	debug  = flag.Bool("debug", true, "print debug logs")
	logger = logr.New()
)

func main() {
	flag.Parse()
	if *debug {
		logr.SetLevel(slog.LevelDebug)
	}

	urlEnv := os.Getenv("URL")
	if urlEnv == "" {
		urlEnv = "localhost:8088"
	}

	room := os.Getenv("ROOM")
	if room == "" {
		room = "test"
	}

	url := url.URL{Scheme: "ws", Host: urlEnv, Path: "/ws"}

	var transcriber stt.Transcriber
	var translator stt.Translator
	var err error

	transcriptionService := os.Getenv("TRANSCRIPTION_SERVICE")
	if transcriptionService != "" {
		transcriber, err = stt.NewHTTPTranscriber(transcriptionService)
		if err != nil {
			logger.Fatal(err, "error creating http api")
		}
	}

	translatorService := os.Getenv("TRANSLATOR_SERVICE")
	if translatorService != "" {
		translator, err = stt.NewHTTPTranslator(translatorService)
		if err != nil {
			logger.Fatal(err, "error creating http api")
		}
	}

	transcriptionStream := make(chan stt.Transcription, 100)

	sttEngine, err := stt.New(stt.EngineParams{
		Transcriber: transcriber,
		Translator:  translator,
		OnDocumentUpdate: func(document stt.Document) {
			// Only send the last transcript.
			transcriptionStream <- document.Transcriptions[len(document.Transcriptions)-1]
		},
	})
	if err != nil {
		logger.Fatal(err, "error creating saturday client")
	}

	sc, err := client.NewSaturdayClient(client.SaturdayConfig{
		Url:                 url,
		Room:                room,
		SttEngine:           sttEngine,
		TranscriptionStream: transcriptionStream,
	})
	if err != nil {
		logger.Fatal(err, "error creating saturday client")
	}

	logger.Info("Starting Saturday Client...")

	if err := sc.Start(); err != nil {
		logger.Fatal(err, "error starting Saturday Client")
	}
}
