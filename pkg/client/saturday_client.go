package client

import (
	"errors"
	"net/url"

	logr "github.com/ajbouh/bridge/pkg/log"
	stt "github.com/ajbouh/bridge/pkg/stt"

	"github.com/pion/webrtc/v3"
)

var Logger = logr.New()

type SaturdayConfig struct {
	// ION room name to connect to
	Room string
	// URL for websocket server
	Url url.URL
	// STT engine to generate transcriptions
	SttEngine *stt.Engine

	// channel used to send transcription segments over the data channel
	// any transcription segment sent on this channel with be sent over the data channel
	DocumentStream chan stt.Document
}

type SaturdayClient struct {
	ws     *SocketConnection
	rtc    *RTCConnection
	config SaturdayConfig
	ae     *AudioEngine
}

func NewSaturdayClient(config SaturdayConfig) (*SaturdayClient, error) {
	// TODO allow this to be nil and just disable transcriptions in that case
	if config.SttEngine == nil {
		return nil, errors.New("SttEngine cannot be nil")
	}
	ae, err := NewAudioEngine(config.SttEngine)
	if err != nil {
		return nil, err
	}

	ws := NewSocketConnection(config.Url)

	rtc, err := NewRTCConnection(RTCConnectionParams{
		trickleFn: func(candidate *webrtc.ICECandidate, target int) error {
			return ws.SendTrickle(candidate, target)
		},
		rtpChan:        ae.RtpIn(),
		documentStream: config.DocumentStream,
		mediaIn:        ae.MediaOut(),
	})
	if err != nil {
		return nil, err
	}

	s := &SaturdayClient{
		ws:     ws,
		rtc:    rtc,
		config: config,
		ae:     ae,
	}

	s.ws.SetOnOffer(s.OnOffer)
	s.ws.SetOnAnswer(s.OnAnswer)
	s.ws.SetOnTrickle(s.rtc.OnTrickle)

	return s, nil
}

func (s *SaturdayClient) OnAnswer(answer webrtc.SessionDescription) error {
	return s.rtc.SetAnswer(answer)
}

func (s *SaturdayClient) OnOffer(offer webrtc.SessionDescription) error {
	ans, err := s.rtc.OnOffer(offer)
	if err != nil {
		Logger.Error(err, "error getting answer")
		return err
	}

	return s.ws.SendAnswer(ans)
}

func (s *SaturdayClient) Start() error {
	if err := s.ws.Connect(); err != nil {
		Logger.Error(err, "error connecting to websocket")
		return err
	}
	offer, err := s.rtc.GetOffer()
	if err != nil {
		Logger.Error(err, "error getting intial offer")
	}
	if err := s.ws.Join(s.config.Room, offer); err != nil {
		Logger.Error(err, "error joining room")
		return err
	}

	s.ae.Start()

	s.ws.WaitForDone()
	Logger.Info("Socket done goodbye")
	return nil
}
