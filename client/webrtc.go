package main

import (
	"encoding/json"
	"fmt"
	"github.com/pion/webrtc/v3"
	"log/slog"
	"path"
	"path/filepath"
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

func (c *WebrtcConn) CreateDataChannel(runType action, flags *Flags) (*webrtc.DataChannel, error) {

	dataChannel, err := c.PeerConnection.CreateDataChannel("dataChannel", nil)
	if err != nil {
		return nil, err
	}

	switch runType {
	case Sender:
		dataChannel.OnOpen(func() {
			fmt.Println("Connection to collector established")
			handleFileSending(dataChannel, flags)
		})
	case Collector:
		dataChannel.OnOpen(func() {
			fmt.Println("Connection to sender established")
		})
	}
	return dataChannel, nil
}

func (c *WebrtcConn) HandleFileReception(d *webrtc.DataChannel, flags *Flags) {
	fmt.Println("handling file reception")
	var metadata FileMetadata

	//TODO: provide some kind of progress based on the number of chunks received
	receivedChunks := make(map[int][]byte)

	c.PeerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			// Assume the first message is metadata
			if metadata.FileSize == 0 && !msg.IsString {
				var err error
				metadata, err = unmarshallMetadata(msg.Data)
				if err != nil {
					slog.Error("Error parsing metadata", "Metadata message", msg.Data, "error", err.Error())
					return
				}
				fmt.Printf("receiving file: %s, size: %d bytes\n", metadata.FileName, metadata.FileSize)
			} else if msg.IsString && string(msg.Data) == "done" { //verify file and request retransmission of chunks if required
				if len(receivedChunks) == metadata.NumChunks {
					path, ok := ensureDirExists("") //TODO: add path as a this as a flag
					if !ok {
						slog.Error("unable to write file", "path", path)
						return
					}
					fp := filepath.Join(path, metadata.FileName) //TODO: add flag for filename and handle duplicates
					if err := writeToFile(fp, receivedChunks, metadata.NumChunks); err != nil {
						slog.Error("unable to write file", "error", err)
						return
					}
				}
				missingSeq, ok := checkForMissingChunks(receivedChunks, metadata.NumChunks)
				if !ok {
					slog.Info("file has missing data in sequence, requesting resend of data")
					if err := requestMissingChunks(d, missingSeq); err != nil {
						slog.Error("attempting to request a retry of chunks failed", "error", err)
						return
					}
				}
			} else { //must be file chunk
				packet, err := unmarshallFilePacket(msg.Data)
				if err != nil {
					slog.Error("Error parsing metadata", "Metadata message", msg.Data, "error", err.Error())
					return
				}
				receivedChunks[packet.SequenceNumber] = packet.Data
			}

		})
	})
}

func (c *WebrtcConn) HandleRetransmission(d *webrtc.DataChannel, flags *Flags) {
	fmt.Println("handling file retransmission messages")
	d.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Received message: %s\n", string(msg.Data))
		request, err := unmashallMissingPacketRequest(msg.Data)
		if err != nil {
			slog.Error("Error parsing Missing packet request", "error", err.Error())
			return
		}

		for _, seq := range request.MissingSequences {
			packet, ok := SerialisedChunks[seq]
			if !ok {
				fp := path.Clean(flags.InputFile)
				err = sequenceFile(fp, flags.ChunkSize)
				if err != nil {
					slog.Error("error resequencing file", "error", err)
					return
				}
				packet, ok = SerialisedChunks[seq]
				if !ok {
					slog.Error("rerequest of a chunk that does not exist")
					return
				}
			}

			packetBytes, err := json.Marshal(packet)
			if err != nil {
				errMsg := fmt.Sprintf("error serializing packet sequence %d: %v", seq, err)
				slog.Error("error sending file", "error", errMsg)
			}
			d.Send(packetBytes)
			slog.Info("retransmission request fulfilled", "seq", seq)
		}
		if err = d.SendText("done"); err != nil {
			slog.Error("error sending done message", "error", err.Error())
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
