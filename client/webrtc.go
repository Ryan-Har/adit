package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type WebrtcConn struct {
	*webrtc.PeerConnection
}

func CreatePeerConnection(additionalStunServer string) (*WebrtcConn, error) {
	stunURLs := []string{"stun:stun.l.google.com:19302",
		"stun:stun1.l.google.com:19302",
		"stun:stun2.l.google.com:19302",
		"stun:stun3.l.google.com:19302",
		"stun:stun4.l.google.com:19302",
	}

	if additionalStunServer != "" {
		stunURLs = append(stunURLs, additionalStunServer)
	}

	config := webrtc.Configuration{
		ICETransportPolicy: webrtc.ICETransportPolicyAll,
		ICEServers: []webrtc.ICEServer{
			{
				URLs: stunURLs,
			},
		},
	}
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}
	return &WebrtcConn{peerConnection}, nil
}

func (c *WebrtcConn) CreateDataChannel(runType action, flags *Flags, wg *sync.WaitGroup) (*webrtc.DataChannel, error) {

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
		dataChannel.OnClose(func() {
			fmt.Println("File recipient saved file, connection closed")
			wg.Done()
		})
	case Collector:
		dataChannel.OnOpen(func() {
			fmt.Println("Connection to sender established")
		})
	}
	return dataChannel, nil
}

func (c *WebrtcConn) HandleFileReception(d *webrtc.DataChannel, flags *Flags, wg *sync.WaitGroup) {
	var metadata FileMetadata
	var bytesReceived int64

	c.PeerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			// Assume the first message is metadata
			if metadata.FileSize == 0 && !msg.IsString {
				var err error
				metadata, err = unmarshallMetadata(msg.Data)
				if err != nil {
					slog.Error("Error parsing metadata", "Metadata message", msg.Data, "error", err.Error())
					wg.Done()
				}
				fmt.Printf("receiving file: %s, size: %d bytes\n", metadata.FileName, metadata.FileSize)
				wg.Add(1)
				go displayTransferPercentage(&bytesReceived, metadata.FileSize, wg)
			} else if msg.IsString && string(msg.Data) == "done" { //verify file and request retransmission of chunks if required
				if len(SerialisedChunks) == metadata.NumChunks {
					var fp string
					if flags.OutputFileName == "" {
						fp = filepath.Join(flags.OutputPath, metadata.FileName)
					} else {
						fp = filepath.Join(flags.OutputPath, flags.OutputFileName)
					}
					if err := writeToFile(fp, metadata.NumChunks); err != nil {
						slog.Error("unable to write file", "error", err)
						wg.Done()
					}
					c.PeerConnection.Close()
					wg.Done()
				}
				missingSeq, ok := checkForMissingChunks(metadata.NumChunks)
				if !ok {
					slog.Info("file has missing data in sequence, requesting resend of data")
					if err := requestMissingChunks(d, missingSeq); err != nil {
						slog.Error("attempting to request a retry of chunks failed", "error", err)
						wg.Done()
					}
				}
			} else { //must be file chunk
				packet, err := unmarshallFilePacket(msg.Data)
				if err != nil {
					slog.Error("Error parsing metadata", "Metadata message", msg.Data, "error", err.Error())
					wg.Done()
				}
				SerialisedChunks[packet.SequenceNumber] = packet
				bytesReceived += int64(len(packet.Data))
			}

		})
	})
}

func (c *WebrtcConn) HandleRetransmission(d *webrtc.DataChannel, flags *Flags) {
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
				err = reSequenceFile(fp, flags.ChunkSize)
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

func (c *WebrtcConn) HandleChanges(ws *Socket, endWG *sync.WaitGroup) {
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
			ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		}
		if state == webrtc.PeerConnectionStateFailed {
			slog.Error("Unable to establish connection to peer")
			//close if we lose connection
			endWG.Done()
		}
	})

	// Log ICE candidate gathering state changes
	c.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		slog.Info("ICE Gathering State has changed", "state", state.String())
	})
}
