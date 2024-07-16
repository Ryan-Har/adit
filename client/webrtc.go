package main

import (
	"fmt"
	"github.com/pion/webrtc/v3"
)

type WebrtcConn struct {
	*webrtc.PeerConnection
}

func CreatePeerConnection() (*WebrtcConn, error) {
	config := webrtc.Configuration{}
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

	dataChannel.OnOpen(func() {
		fmt.Println("Data channel is open")
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Received message: %s\n", string(msg.Data))
	})

	return dataChannel, nil
}

func (c *WebrtcConn) HandleDataChannel() {
	c.PeerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnOpen(func() {
			fmt.Println("Data channel is open")
		})

		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Received message: %s\n", string(msg.Data))
		})
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
