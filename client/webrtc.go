package main

import (
	"fmt"
	"github.com/pion/webrtc/v3"
	"log/slog"
	//"time"
)

type WebrtcConn struct {
	*webrtc.PeerConnection
}

func CreatePeerConnection() (*WebrtcConn, error) {
	config := webrtc.Configuration{
		ICETransportPolicy: webrtc.ICETransportPolicyAll,
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}
	return &WebrtcConn{peerConnection}, nil
}

func (c *WebrtcConn) CreateDataChannel() (*webrtc.DataChannel, error) {

	dataChannel, err := c.PeerConnection.CreateDataChannel("dataChannel", nil)
	if err != nil {
		return nil, err
	}

	return dataChannel, nil
}

func (c *WebrtcConn) HandleDataChannel(runType action) {
	c.PeerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		switch runType {
		case Sender:
			d.OnOpen(func() {
				fmt.Println("Connection to collector established")
			})

			d.OnMessage(func(msg webrtc.DataChannelMessage) {
				fmt.Printf("Received message: %s\n", string(msg.Data))
			})
		case Collector:
			d.OnOpen(func() {
				fmt.Println("Connection to sender established")
			})

			d.OnMessage(func(msg webrtc.DataChannelMessage) {
				fmt.Printf("Received message: %s\n", string(msg.Data))
			})
		}

	})
}

func (c *WebrtcConn) CreateOffer() (*webrtc.SessionDescription, error) {
	offer, err := c.PeerConnection.CreateOffer(nil)
	if err != nil {
		return nil, err
	}

	err = c.PeerConnection.SetLocalDescription(offer)
	if err != nil {
		return nil, err
	}

	return &offer, nil
}

func (c *WebrtcConn) SetRemoteDescription(sdp webrtc.SessionDescription) error {
	return c.PeerConnection.SetRemoteDescription(sdp)
}

func (c *WebrtcConn) CreateAnswer() (*webrtc.SessionDescription, error) {
	answer, err := c.PeerConnection.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}

	err = c.PeerConnection.SetLocalDescription(answer)
	if err != nil {
		return nil, err
	}

	return &answer, nil
}

func (c *WebrtcConn) HandleChanges(ws *Socket) {
	// Log ICE connection state changes
	c.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		slog.Info("ICE Connection State has changed", "state", state.String())
	})

	// Log signaling state changes
	c.OnSignalingStateChange(func(state webrtc.SignalingState) {
		slog.Info("Signaling State has changed", "state", state.String())
	})

	// Log connection state changes
	c.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		slog.Info("PeerConnection State has changed", "state", state.String())
		if state == webrtc.PeerConnectionStateConnected {
			//cleanup websocket connection
			ws.Close()
		}
		if state == webrtc.PeerConnectionStateFailed {
			slog.Error("Unable to establish connection to peer")
		}
	})

	// Log ICE candidate gathering state changes
	c.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		slog.Info("ICE Gathering State has changed", "state", state.String())
	})
}
