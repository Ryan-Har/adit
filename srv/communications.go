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
}

func (p *Peer) handleConnection() {
	defer p.Close()
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
		case websocket.BinaryMessage:
			slog.Info("message received from", "remoteddr", p.RemoteAddr())
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

		// p.sendMessage(&Message{
		// 	MessageType: "success",
		// 	Content:     "successfully added answer sdp",
		// })

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

func (p *Peer) sendSuccess() {
	msg := &Message{
		MessageType: "success",
	}
	p.sendMessage(msg)
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
