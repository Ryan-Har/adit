package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"log/slog"
)

type Message struct {
	MessageType string `json:"messagetype"`
	Phrase      string `json:"phrase"`
	Content     any    `json:"content"`
}

type Peers struct {
	PeerSender    Peer
	PeerCollector Peer
	OfferSdp      webrtc.SessionDescription
	AnswerSdp     webrtc.SessionDescription
}

type Peer struct {
	*websocket.Conn
	Phrase string
}

func (p *Peer) handleConnection() {
	defer func() {
		p.Close()
		peers, ok := ongoingSessions[p.Phrase]
		if !ok {
			fmt.Println("session already removed")
			return // Session already removed
		}
		blankPeer := Peer{}

		if peers.PeerSender == *p {
			peers.PeerSender = blankPeer
		} else {
			peers.PeerCollector = blankPeer
		}

		// // Check if it's the last peer in the session
		if peers.PeerSender == blankPeer && peers.PeerCollector == blankPeer {
			delete(ongoingSessions, p.Phrase)
		}
	}()

	for {
		// Read message
		messageType, message, err := p.ReadMessage()
		if err != nil {
			fmt.Println("Read error:", err)
			break
		}

		switch messageType {
		case websocket.CloseMessage:
			slog.Info("websocket close message from", "remoteAddr", p.RemoteAddr())
		case websocket.TextMessage:
			slog.Info("message received from", "remoteddr", p.RemoteAddr())
			p.handleTextMessage(message)
			slog.Info("response generated for", "remoteaddr", p.RemoteAddr())
		}
	}

}

func (p *Peer) handleTextMessage(message []byte) {
	msg := &Message{}

	err := json.Unmarshal(message, &msg)
	if err != nil {
		p.sendMessage(&Message{
			MessageType: "error",
			Content:     err,
		})
		return
	}
	slog.Info("text message handled", "type", msg.MessageType, "message", msg.Content)

	switch msg.MessageType {
	case "ping":
		p.sendMessage(&Message{
			MessageType: "pong",
		})
		return
	case "offer":
		words, err := GetNumberOfWords(5)
		if err != nil {
			p.sendMessage(&Message{
				MessageType: "error",
				Content:     err,
			})
			return
		}
		slog.Info("word phrase generated", "words", words)

		sdpString, ok := msg.Content.(string)
		if !ok {
			p.sendMessage(&Message{
				MessageType: "error",
				Content:     errors.New("error reading the sdp string provided"),
			})
			return
		}

		p.Phrase = words
		ongoingSessions[words] = &Peers{
			PeerSender: *p,
			OfferSdp: webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  sdpString,
			},
		}

		p.sendMessage(&Message{
			MessageType: "phrase create",
			Phrase:      words,
			Content:     words,
		})
		return
	case "get offer":
		if msg.Phrase == "" {
			p.sendMessage(&Message{
				MessageType: "error",
				Content:     errors.New("phrase is empty, cannot collect without phrase"),
			})
			return
		}
		_, ok := ongoingSessions[msg.Phrase]
		if !ok {
			p.sendMessage(&Message{
				MessageType: "error",
				Content:     errors.New("phrase does not exist"),
			})
			return
		}
		p.Phrase = msg.Phrase
		ongoingSessions[msg.Phrase].PeerCollector = *p
		p.sendMessage(&Message{
			MessageType: "offer",
			Phrase:      msg.Phrase,
			Content:     ongoingSessions[msg.Phrase].OfferSdp.SDP,
		})
		return
	case "answer":
		if msg.Phrase == "" {
			p.sendMessage(&Message{
				MessageType: "error",
				Content:     errors.New("phrase is empty, cannot collect without phrase"),
			})
			return
		}

		sdpString, ok := msg.Content.(string)
		if !ok {
			p.sendMessage(&Message{
				MessageType: "error",
				Content:     errors.New("error reading the sdp string provided"),
			})
			return
		}

		ongoingSessions[msg.Phrase].AnswerSdp = webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  sdpString,
		}

		ongoingSessions[msg.Phrase].PeerSender.sendMessage(&Message{
			MessageType: "answer",
			Phrase:      msg.Phrase,
			Content:     ongoingSessions[msg.Phrase].AnswerSdp.SDP,
		})
		return
	case "ice candidate":
		if msg.Phrase == "" {
			p.sendMessage(&Message{
				MessageType: "error",
				Content:     errors.New("phrase is empty, cannot collect without phrase"),
			})
			return
		}

		if ongoingSessions[msg.Phrase].PeerSender == *p {
			fmt.Println("sending candidate to collector")
			ongoingSessions[msg.Phrase].PeerCollector.sendMessage(msg)
		} else {
			fmt.Println("sending candidate to sender")
			ongoingSessions[msg.Phrase].PeerSender.sendMessage(msg)
		}
		return
	}
	p.sendMessage(&Message{
		MessageType: "error",
		Content:     fmt.Errorf("Message type %v is not understood", msg.MessageType),
	})
}

func (p *Peer) sendMessage(m *Message) {
	if m.MessageType == "error" {
		slog.Error("error in response message", "error", m.Content)
	}

	jsonBytes, err := json.Marshal(m)
	if err != nil {
		slog.Error("error marshalling response message", "message", m, "error", err)
	}
	err = p.WriteMessage(websocket.TextMessage, jsonBytes)
	if err != nil {
		slog.Error("Write error:", "error", err)
	}
	slog.Info("message sent to", "remoteaddr", p.RemoteAddr(), "message", m)
}
