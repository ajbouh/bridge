package main

import (
	"flag"
	"net/url"
	"os"
	"time"

	"github.com/ajbouh/bridge/pkg/assistant"
	"github.com/ajbouh/bridge/pkg/chat"
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

	clientStream := make(chan stt.Document, 100)
	docListeners := []chan stt.Document{
		clientStream,
	}
	transcriptionStream := make(chan *stt.Transcription, 100)
	audioStream := make(chan *stt.CapturedAudio, 100)

	go func() {
		document := stt.Document{
			StartedAt: time.Now().Unix(),
		}
		for transcription := range transcriptionStream {
			document.Transcriptions = append(document.Transcriptions, transcription)
			snapshot := stt.Document{
				StartedAt:      document.StartedAt,
				Transcriptions: append([]*stt.Transcription{}, document.Transcriptions...),
			}
			for _, o := range docListeners {
				o <- snapshot
			}
		}
	}()

	var transcriber *stt.HttpTranscriber
	var translator *stt.HttpTranslator
	var err error

	transcriptionService := os.Getenv("TRANSCRIPTION_SERVICE")
	if transcriptionService != "" {
		transcriber, err = stt.NewHTTPTranscriber(transcriptionService)
		if err != nil {
			logger.Fatal(err, "error creating http api")
		}
		go transcriber.Run(transcriptionStream, audioStream)
	}

	translatorService := os.Getenv("TRANSLATOR_SERVICE")
	if translatorService != "" {
		translator, err = stt.NewHTTPTranslator(translatorService)
		if err != nil {
			logger.Fatal(err, "error creating http api")
		}
		listener := make(chan stt.Document, 100)
		docListeners = append(docListeners, listener)

		go translator.Run(transcriptionStream, listener)
	}


	sttEngine, err := stt.New(func(a *stt.CapturedAudio) {
		audioStream <- a
	})
	if err != nil {
		logger.Fatal(err, "error creating saturday client")
	}

	sc, err := client.NewSaturdayClient(client.SaturdayConfig{
		Url:            url,
		Room:           room,
		SttEngine:      sttEngine,
		DocumentStream: clientStream,
	})
	if err != nil {
		logger.Fatal(err, "error creating saturday client")
	}

	logger.Info("Starting Saturday Client...")

	if err := sc.Start(); err != nil {
		logger.Fatal(err, "error starting Saturday Client")
	}
}
