package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"encoding/base64"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type Socket struct {
	*websocket.Conn
	*ConnectionItems
}

type ConnectionItems struct {
	offerSDP  webrtc.SessionDescription
	answerSDP webrtc.SessionDescription
	Phrase    string
}

type Message struct {
	MessageType string `json:"messagetype"`
	Phrase      string `json:"phrase"`
	Content     any    `json:"content"`
}

var SDPTypeMap = map[string]webrtc.SDPType{
	"answer": webrtc.SDPTypeAnswer,
	"offer":  webrtc.SDPTypeOffer,
}

func WebsocketConnect(url url.URL) (*Socket, error) {

	dialer := websocket.DefaultDialer

	conn, _, err := dialer.Dial(url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error connecting to relay server at %s", url.String())
	}

	s := &Socket{
		conn,
		&ConnectionItems{},
	}

	if err := s.ping(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Socket) ping() error {
	msg := &Message{
		MessageType: "ping",
	}
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = s.WriteMessage(websocket.TextMessage, jsonBytes)
	if err != nil {
		return err
	}

	_, receivedMessage, err := s.ReadMessage()
	if err != nil {
		return fmt.Errorf("error reading message %v", err.Error())
	}

	response := &Message{}
	err = json.Unmarshal(receivedMessage, &response)
	if err != nil {
		return fmt.Errorf("error unmarshalling response %v", err.Error())
	}

	if response.MessageType != "pong" {
		return errors.New("received message but was not pong")
	}

	return nil
}

func (s *Socket) keepAlive() {
	msg := &Message{
		MessageType: "ping",
	}
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		slog.Error(err.Error())
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		err := s.WriteMessage(websocket.TextMessage, jsonBytes)
		if err != nil {
			slog.Error("error sending keepalive message to websocket server")
		} else {
			slog.Info("successfully sent keepalive to websocket server")
		}
	}
}

func (s *Socket) SendWebrtcSessionDescription(sdp *webrtc.SessionDescription) error {
	msg := &Message{
		MessageType: sdp.Type.String(),
		Phrase:      s.Phrase,
		Content:     sdp.SDP,
	}
	return s.marshalAndSend(msg)
}

func (s *Socket) SendIceCandidate(ic *webrtc.ICECandidate) error {
	candidateJson := ic.ToJSON()
	candidateStr, err := json.Marshal(candidateJson)
	encoded := base64.StdEncoding.EncodeToString([]byte(candidateStr))
	if err != nil {
		return err
	}
	msg := &Message{
		MessageType: "ice candidate",
		Phrase:      s.Phrase,
		Content:     encoded,
	}
	return s.marshalAndSend(msg)
}

func (s *Socket) marshalAndSend(msg *Message) error {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = s.WriteMessage(websocket.TextMessage, jsonBytes)
	if err != nil {
		return err
	}
	return nil
}

func (s *Socket) GetOffer() error {
	message := &Message{
		MessageType: "get offer",
		Phrase:      s.Phrase,
	}

	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	err = s.WriteMessage(websocket.TextMessage, jsonBytes)
	if err != nil {
		return err
	}
	return nil
}

func (s *Socket) HandleIncomingMessages(peerConn *WebrtcConn) {
	for {
		_, receivedMessage, err := s.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
				slog.Error("Error reading from websocket", "error", err.Error())
				os.Exit(1)
			} else {
				break
			}
		}
		msg := &Message{}
		err = json.Unmarshal(receivedMessage, &msg)
		if err != nil {
			slog.Error("Error unmarshalling message", "error", err.Error())
		}

		switch msg.MessageType {
		case "phrase create":
			s.Phrase = msg.Phrase
			//notify user so it can be sent to sender
			fmt.Println("Phrase generated for file transfer:", msg.Phrase)
		case "answer":
			answerSDP, err := msg.toSessionDescription()
			if err != nil {
				slog.Error(err.Error())
			}
			s.answerSDP = *answerSDP
			if err := peerConn.SetRemoteDescription(*answerSDP); err != nil {
				slog.Error("unable to set remote description", "error", err.Error())
			}
		case "offer":
			offerSDP, err := msg.toSessionDescription()
			if err != nil {
				slog.Error(err.Error())
			}
			if err := peerConn.SetRemoteDescription(*offerSDP); err != nil {
				slog.Error("unable to set remote description", "error", err.Error())
			}
			s.offerSDP = *offerSDP
		case "ice candidate":
			candidate, err := msg.toIceCandidate()
			if err != nil {
				slog.Error("error getting ice candidate", "error", err)
			}
			peerConn.AddICECandidate(*candidate)
		case "error":
			slog.Error("error occured when establising connection to peer, please try again")
		case "pong":
			slog.Info("keepalive successful")
		}
	}
}

// this assumes it's a valid SDP
func (m *Message) toSessionDescription() (*webrtc.SessionDescription, error) {
	sdpString, ok := m.Content.(string)
	if !ok {
		return nil, errors.New("error reading the sdp string from response")
	}

	return &webrtc.SessionDescription{
		Type: SDPTypeMap[m.MessageType],
		SDP:  sdpString,
	}, nil
}

func (m *Message) toIceCandidate() (*webrtc.ICECandidateInit, error) {
	iceString, ok := m.Content.(string)
	if !ok {
		return nil, errors.New("error reading the candidate string from response")
	}

	decoded, err := base64.StdEncoding.DecodeString(iceString)
	if err != nil {
		return nil, err
	}

	var candidate webrtc.ICECandidateInit

	err = json.Unmarshal([]byte(decoded), &candidate)
	if err != nil {
		return nil, err
	}

	return &candidate, nil
}
