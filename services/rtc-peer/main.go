package main

import (
	"context"
	"flag"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ajbouh/bridge/pkg/assistant"
	logr "github.com/ajbouh/bridge/pkg/log"
	"github.com/ajbouh/bridge/pkg/router"
	"github.com/ajbouh/bridge/pkg/transcriber"
	"github.com/ajbouh/bridge/pkg/translator"
	"github.com/ajbouh/bridge/pkg/vad"
	"github.com/ajbouh/bridge/pkg/webrtcpeer"

	"golang.org/x/exp/slog"
)

var (
	debug  = flag.Bool("debug", true, "print debug logs")
	logger = logr.New()
)

func getenvPrefixMap(prefix string) map[string]string {
	m := map[string]string{}
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, prefix) {
			k, v, _ := strings.Cut(entry, "=")
			m[strings.TrimPrefix(k, prefix)] = v
		}
	}

	return m
}

func main() {
	flag.Parse()
	if *debug {
		logr.SetLevel(slog.LevelDebug)
	}

	r := router.New(context.Background())
	r.Start()

	transcriptionService := os.Getenv("BRIDGE_TRANSCRIPTION")
	if transcriptionService != "" {
		fn, err := transcriber.New(transcriptionService)
		if err != nil {
			logger.Fatal(err, "error creating transcriber")
		}
		r.InstallMiddleware(fn)
	}

	translators := getenvPrefixMap("BRIDGE_TRANSLATOR_")
	for translatorConfig, translatorService := range translators {
		modality, targetLanguagesStr, _ := strings.Cut(translatorConfig, "_")
		targetLanguages := strings.Split(targetLanguagesStr, "_")
		var useAudio bool
		if modality == "audio" {
			useAudio = true
		}
		fn, err := translator.New(translatorService, useAudio, targetLanguages[0], targetLanguages[1:]...)
		if err != nil {
			logger.Fatal(err, "error creating translator")
		}

		r.InstallMiddleware(fn)
	}

	assistants := getenvPrefixMap("BRIDGE_ASSISTANT_")
	for assistantName, assistantService := range assistants {
		r.InstallMiddleware(assistant.New(assistantName, assistantService))
	}

	r.InstallMiddleware(vad.New(vad.Config{
		SampleRate:   16000,
		SampleWindow: 24 * time.Second,
	}))

	webrtcpeerURL := os.Getenv("BRIDGE_WEBRTC_URL")
	if webrtcpeerURL != "" {
		room := os.Getenv("BRIDGE_WEBRTC_ROOM")
		if room == "" {
			room = "test"
		}

		url := url.URL{Scheme: "ws", Host: webrtcpeerURL, Path: "/ws"}
		r.InstallMiddleware(webrtcpeer.New(url, room))
	}

	r.WaitForDone()
}
