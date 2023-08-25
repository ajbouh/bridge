package main

import (
	"fmt"
	"net/http"

	"github.com/ajbouh/bridge/services/rtc/internal/ws"

	"github.com/gorilla/websocket"
	log "github.com/pion/ion-sfu/pkg/logger"
	"github.com/pion/ion-sfu/pkg/sfu"
	"github.com/sourcegraph/jsonrpc2"
	websocketjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"
	"github.com/spf13/viper"
)

var (
	conf   = sfu.Config{}
	logger = log.New()
)

func main() {

	logger.Info("Starting S.A.T.U.R.D.A.Y RTC server...")

	// build + start sfu

	viper.SetConfigFile("./config.toml")
	viper.SetConfigType("toml")
	err := viper.ReadInConfig()
	if err != nil {
		logger.Error(err, "error reading config")
		panic(err)
	}

	err = viper.Unmarshal(&conf)
	if err != nil {
		logger.Error(err, "error unmarshalling config")
		panic(err)
	}

	// start websocket server

	sfu.Logger = logger
	s := sfu.NewSFU(conf)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// http.Handle("/", http.FileServer(http.Dir("./web")))
	http.Handle("/", http.FileServer(http.Dir("./ui")))

	// Set up a handler function for the `/ws` path
	http.HandleFunc("/ws", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Upgrading conn...")
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			panic(err)
		}
		defer c.Close()

		p := ws.NewConnection(sfu.NewPeer(s), logger)
		defer p.Close()

		jc := jsonrpc2.NewConn(r.Context(), websocketjsonrpc2.NewObjectStream(c), p)
		<-jc.DisconnectNotify()

	}))

	port := 8088
	// transcriptionService := os.Getenv("TRANSCRIPTION_SERVICE")
	// translatorService := os.Getenv("TRANSLATOR_SERVICE")
	// if transcriptionService != "" {
	// 	go func() {
	// 		transcriber, err := stt.NewHTTPTranscriber(transcriptionService)
	// 		if err != nil {
	// 			logger.Error(err, "error creating http api")
	// 			panic(err)
	// 		}

	// 		translator, err := stt.NewHTTPTranslator(translatorService)
	// 		if err != nil {
	// 			logger.Error(err, "error creating http api")
	// 			panic(err)
	// 		}

	// 		err = RunNewRoomTranscriber(
	// 			transcriber,
	// 			translator,
	// 			url.URL{Scheme: "ws", Host: fmt.Sprintf("localhost:%d", port), Path: "/ws"},
	// 			"test",
	// 		)
	// 		if err != nil {
	// 			logger.Error(err, "error creating transcriber")
	// 			panic(err)
	// 		}
	// 	}()
	// }

	err = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil)
	if err != nil {
		fmt.Println(err)
	}
}

// func RunNewRoomTranscriber(transcriber stt.Transcriber, translator stt.Translator, url url.URL, room string) error {
// 	transcriptionStream := make(chan stt.Document, 100)
// 	sttEngine, err := stt.New(stt.EngineParams{
// 		Transcriber: transcriber,
// 		Translator:  translator,
// 		OnDocumentUpdate: func(document stt.Document) {
// 			transcriptionStream <- document
// 		},
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	sc, err := client.NewSaturdayClient(client.SaturdayConfig{
// 		Url:                 url,
// 		Room:                room,
// 		SttEngine:           sttEngine,
// 		TranscriptionStream: transcriptionStream,
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	return sc.Start()
// }
