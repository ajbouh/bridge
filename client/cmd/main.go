package main

import (
	"flag"
	"net/url"
	"os"

	"github.com/GRVYDEV/S.A.T.U.R.D.A.Y/client"
	logr "github.com/GRVYDEV/S.A.T.U.R.D.A.Y/log"
	shttp "github.com/GRVYDEV/S.A.T.U.R.D.A.Y/stt/backends/http"
	// whisper "github.com/GRVYDEV/S.A.T.U.R.D.A.Y/stt/backends/whisper.cpp"
	stt "github.com/GRVYDEV/S.A.T.U.R.D.A.Y/stt/engine"

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
	var err error

	transcriptionService := os.Getenv("TRANSCRIPTION_SERVICE")
	if transcriptionService != "" {
		transcriptionUrl := transcriptionService + room + "/transcribe" // Replace with the appropriate API URL

		transcriber, err = shttp.New(transcriptionUrl)
		if err != nil {
			logger.Fatal(err, "error creating http api")
		}

		// } else {
		// 	// FIXME read from env
		// 	transcriber, err = whisper.New("models/ggml-base.en.bin")
		// 	if err != nil {
		// 		logger.Fatal(err, "error creating whisper model")
		// 	}
	}

	transcriptionStream := make(chan stt.Document, 100)

	documentComposer := stt.NewDocumentComposer()
	// documentComposer.FilterSegment(func(ts stt.TranscriptionSegment) bool {
	// 	return ts.Text[0] == '.' || strings.ContainsAny(ts.Text, "[]()")
	// })

	sttEngine, err := stt.New(stt.EngineParams{
		Transcriber:      transcriber,
		DocumentComposer: documentComposer,
		UseVad:           true,
	})

	sc, err := client.NewSaturdayClient(client.SaturdayConfig{
		Room:                room,
		Url:                 url,
		SttEngine:           sttEngine,
		TranscriptionStream: transcriptionStream,
	})
	if err != nil {
		logger.Fatal(err, "error creating saturday client")
	}

	sttEngine.OnDocumentUpdate(func(document stt.Document) {
		transcriptionStream <- document
	})

	logger.Info("Starting Saturday Client...")

	if err := sc.Start(); err != nil {
		logger.Fatal(err, "error starting Saturday Client")
	}
}
