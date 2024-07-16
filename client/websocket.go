package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type Socket struct {
	*websocket.Conn
	channel
}

type channel struct {
	receiveChan chan *Message
}

var Phrase string

type Message struct {
	MessageType string `json:"messagetype"`
	Phrase      string `json:"phrase"`
	Content     any    `json:"content"`
}

// type Connection struct {
// 	offerSDP  webrtc.SessionDescription
// 	answerSDP webrtc.SessionDescription
// }

var SDPTypeMap = map[string]webrtc.SDPType{
	"answer": webrtc.SDPTypeAnswer,
	"offer":  webrtc.SDPTypeOffer,
}

// TODO: this contains the outline of websocket client connection logic. It needs to be changed
func WebsocketConnect(url url.URL) (*Socket, error) {

	dialer := websocket.DefaultDialer

	conn, _, err := dialer.Dial(url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error dialling, %w", err)
	}

	chans := &channel{
		receiveChan: make(chan *Message),
	}

	s := &Socket{
		conn,
		*chans,
	}

	if err := s.ping(); err != nil {
		return nil, err
	}

	go s.handleReceives()

	return s, nil
}

// should only be used before channels are set up, this is why it is not exported
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

func (s *Socket) handleReceives() {
	for {
		_, receivedMessage, err := s.ReadMessage()
		if err != nil {
			slog.Error("Error reading message", "error", err.Error())
		}
		msg := &Message{}
		err = json.Unmarshal(receivedMessage, &msg)
		if err != nil {
			slog.Error("Error unmarshalling message", "error", err.Error())
		}
		s.receiveChan <- msg
	}
}

func (s *Socket) GetMessage() *Message {
	return <-s.receiveChan
}

func (s *Socket) SendWebrtcSessionDescription(sdp *webrtc.SessionDescription) error {
	msg := &Message{
		MessageType: sdp.Type.String(),
		Phrase:      Phrase,
		Content:     sdp.SDP,
	}
	return s.marshalAndSend(msg)
}

func (s *Socket) SendIceCandidate(ic *webrtc.ICECandidate) error {
	msg := &Message{
		MessageType: "ice candidate",
		Phrase:      Phrase,
		Content:     ic.ToJSON().Candidate,
	}
	return s.marshalAndSend(msg)
}

func (s *Socket) GetMessageWithExpectedType(msgType string) (*Message, bool) {
	msg := s.GetMessage()
	if msgType == "error" && msg.MessageType == "error" {
		return msg, true
	}

	if msg.MessageType == "error" {
		slog.Error("got message which had error", "message content", msg.Content)
		return msg, false
	}

	if msg.MessageType != msgType {
		return msg, false
	}

	return msg, true
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
		Phrase:      Phrase,
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

// this assumes it's a valid SDP
func (m *Message) ToSessionDescription() (*webrtc.SessionDescription, error) {
	sdpString, ok := m.Content.(string)
	if !ok {
		return nil, errors.New("error reading the sdp string from response")
	}

	return &webrtc.SessionDescription{
		Type: SDPTypeMap[m.MessageType],
		SDP:  sdpString,
	}, nil
}

func (m *Message) ToIceCandidate() (*webrtc.ICECandidateInit, error) {
	iceString, ok := m.Content.(string)
	if !ok {
		return nil, errors.New("error reading the sdp string from response")
	}
	var candidate webrtc.ICECandidateInit

	err := json.Unmarshal([]byte(iceString), &candidate)
	if err != nil {
		return nil, err
	}

	return &candidate, nil
}
