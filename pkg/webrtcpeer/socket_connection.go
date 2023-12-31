package webrtcpeer

import (
	"encoding/json"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// JoinConfig allow adding more control to the peers joining a SessionLocal.
type JoinConfig struct {
	// If true the peer will not be allowed to publish tracks to SessionLocal.
	NoPublish bool
	// If true the peer will not be allowed to subscribe to other peers in SessionLocal.
	NoSubscribe bool
	// If true the peer will not automatically subscribe all tracks,
	// and then the peer can use peer.Subscriber().AddDownTrack/RemoveDownTrack
	// to customize the subscrbe stream combination as needed.
	// this parameter depends on NoSubscribe=false.
	NoAutoSubscribe bool
}

// TODO move these to core
// Join message sent when initializing a peer connection
type Join struct {
	SID    string                    `json:"sid"`
	UID    string                    `json:"uid"`
	Offer  webrtc.SessionDescription `json:"offer,omitempty"`
	Config JoinConfig                `json:"config,omitempty"`
}

// Negotiation message sent when renegotiating the peer connection
type Negotiation struct {
	Desc webrtc.SessionDescription `json:"desc"`
}

// Trickle message sent when renegotiating the peer connection
type Trickle struct {
	Target    int                     `json:"target"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type Message[T Join | Negotiation | Trickle] struct {
	Method string `json:"method"`
	Params T      `json:"params"`
}

type SocketConnection struct {
	url  url.URL
	ws   *websocket.Conn
	done chan int

	// called when we get a remote offer
	onOffer func(offer webrtc.SessionDescription) error
	// called when we get a remote answer
	onAnswer func(ans webrtc.SessionDescription) error
	// called when we get a remote candidate
	onTrickle func(candidate webrtc.ICECandidateInit, target int) error

	wsWriteMutex *sync.Mutex
}

func NewSocketConnection(url url.URL) *SocketConnection {
	return &SocketConnection{
		url:          url,
		done:         make(chan int),
		wsWriteMutex: &sync.Mutex{},
	}
}

func (s *SocketConnection) WaitForDone() {
	<-s.done
}

func (s *SocketConnection) SetOnOffer(onOffer func(offer webrtc.SessionDescription) error) {
	s.onOffer = onOffer
}

func (s *SocketConnection) SetOnAnswer(onAnswer func(ans webrtc.SessionDescription) error) {
	s.onAnswer = onAnswer
}

func (s *SocketConnection) SetOnTrickle(onTrickle func(candidate webrtc.ICECandidateInit, target int) error) {
	s.onTrickle = onTrickle
}

func (s *SocketConnection) Connect() error {
	c, _, err := websocket.DefaultDialer.Dial(s.url.String(), nil)
	if err != nil {
		Logger.Error(err, "dial err")
		return err
	}

	s.ws = c
	return nil
}

func (s *SocketConnection) Join(room string, offer webrtc.SessionDescription) error {
	msg := Message[Join]{
		Method: "join",
		Params: Join{
			SID:   room,
			UID:   "SaturdayClient",
			Offer: offer,
		},
	}

	if err := s.sendMessage(msg); err != nil {
		Logger.Errorf(err, "Error sending join message %+v", msg)
		return err
	}

	go s.readMessages()
	return nil
}

func (s *SocketConnection) readMessages() error {
	for {
		_, message, err := s.ws.ReadMessage()
		if err != nil {
			Logger.Error(err, "err reading message")
			s.ws.Close()
			close(s.done)
			return err
		}

		var msg map[string]interface{}

		json.Unmarshal(message, &msg)

		// FIXME handle errors better
		switch msg["method"] {
		case "offer":
			params, ok := msg["params"].(map[string]interface{})
			if !ok {
				Logger.Infof("invalid params for offer %+v", msg["params"])
				continue
			}
			ty, ok := params["type"].(string)
			if !ok {
				Logger.Infof("invalid type for offer %+v", params["type"])
				continue
			}
			sdp, ok := params["sdp"].(string)
			if !ok {
				Logger.Infof("invalid sdp for offer %+v", params["sdp"])
				continue
			}

			offer := webrtc.SessionDescription{Type: webrtc.NewSDPType(ty), SDP: sdp}

			if s.onOffer != nil {
				if err := s.onOffer(offer); err != nil {
					Logger.Errorf(err, "error calling onOffer with offer %+v", offer)
				}
			}
		case "trickle":
			params, ok := msg["params"].(map[string]interface{})
			if !ok {
				Logger.Infof("invalid params for trickle %+v", msg["params"])
				continue
			}

			paramsJson, err := json.Marshal(params)
			if err != nil {
				Logger.Error(err, "error marshalling trickle params")
				continue
			}

			var trickle Trickle

			if err = json.Unmarshal(paramsJson, &trickle); err != nil {
				Logger.Error(err, "error unmarshalling trickle params")
				continue
			}

			if s.onTrickle != nil {
				if err := s.onTrickle(trickle.Candidate, trickle.Target); err != nil {
					Logger.Errorf(err, "error calling onTrickle with candidate %+v", trickle)
				}
			}

		default:
			res, ok := msg["result"].(map[string]interface{})
			if !ok {
				Logger.Infof("got unhandled message: %+v", msg)
				continue
			}
			sdp, ok := res["sdp"].(string)
			if !ok {
				Logger.Infof("invalid sdp for answer %+v", res["sdp"])
				continue
			}
			ty, ok := res["type"].(string)
			if !ok {
				Logger.Infof("invalid sdp type for answer %+v", res["type"])
				continue
			}
			answer := webrtc.SessionDescription{Type: webrtc.NewSDPType(ty), SDP: sdp}

			if s.onAnswer != nil {
				if err := s.onAnswer(answer); err != nil {
					Logger.Errorf(err, "error calling onAnswer with answer %+v", answer)
				}
			}

		}

	}
}

func (s *SocketConnection) SendTrickle(candidate *webrtc.ICECandidate, target int) error {
	if candidate == nil {
		return nil
	}

	msg := Message[Trickle]{
		Method: "trickle",
		Params: Trickle{
			Target:    target,
			Candidate: candidate.ToJSON(),
		},
	}

	Logger.Debug("Sending trickle")

	return s.sendMessage(msg)
}

func (s *SocketConnection) SendAnswer(answer webrtc.SessionDescription) error {
	msg := Message[Negotiation]{
		Method: "answer",
		Params: Negotiation{
			Desc: answer,
		},
	}

	Logger.Debug("Sending answer")

	return s.sendMessage(msg)
}

func (s *SocketConnection) sendMessage(msg any) error {
	s.wsWriteMutex.Lock()
	defer s.wsWriteMutex.Unlock()

	payload, err := json.Marshal(msg)
	if err != nil {
		Logger.Errorf(err, "Error marshaling message to json %+v", msg)
		return err
	}
	Logger.Debugf("Sending message %s", payload)
	if err := s.ws.WriteMessage(websocket.TextMessage, payload); err != nil {
		Logger.Errorf(err, "Error sending websocket message %+v", msg)
		return err
	}
	return nil
}
