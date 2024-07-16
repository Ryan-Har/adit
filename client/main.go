package main

import (
	"fmt"
	"github.com/pion/webrtc/v3"
	"log/slog"
	"os"
	"time"
)

func main() {
	flags, err := GetFlags()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: flags.logLevel})))

	if flags.InputFile != "" {
		slog.Info("input file check", "input file", flags.InputFile)
		inputOperations(flags)
	}

	if flags.CollectCode != "" {
		slog.Info("collection code check", "collect code", flags.CollectCode)
		collectOperations(flags)
	}
}

func inputOperations(flags *Flags) {
	//TODO: Validate file / folder before making expensive network call

	ws, err := WebsocketConnect(*flags.Server)
	if err != nil {
		slog.Error("unable to initialise websocket connection", "error", err.Error())
	}
	defer ws.Close()

	rtc, err := CreatePeerConnection()
	if err != nil {
		slog.Error("unable to create peer connection", "error", err.Error())
	}
	defer rtc.Close()

	// Log ICE connection state changes
	rtc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", state.String())
	})

	// Log signaling state changes
	rtc.OnSignalingStateChange(func(state webrtc.SignalingState) {
		fmt.Printf("Signaling State has changed: %s\n", state.String())
	})

	// Log connection state changes
	rtc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("PeerConnection State has changed: %s\n", state.String())
		if state == webrtc.PeerConnectionStateFailed {
			fmt.Println("Connection has failed. Additional debugging information may be needed.")
		}
	})

	// Log ICE candidate gathering state changes
	rtc.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		fmt.Printf("ICE Gathering State has changed: %s\n", state.String())
	})

	// rtc.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
	// 	fmt.Println("connection state change", rtc.ConnectionState())
	// })

	rtcDataChan, err := rtc.CreateDataChannel()
	if err != nil {
		slog.Error("unable to create data channel", "error", err.Error())
	}
	defer rtcDataChan.Close()

	rtc.HandleDataChannel()

	offerSDP, err := rtc.CreateOffer()
	if err != nil {
		slog.Error("unable to create offer", "error", err.Error())
	}

	if err := ws.SendWebrtcSessionDescription(offerSDP); err != nil {
		slog.Error("unable to send offer", "error", err.Error())
	}

	_, ok := ws.GetMessageWithExpectedType("success")
	if !ok {
		slog.Error("got message with unexpected type")
	}

	msg, ok := ws.GetMessageWithExpectedType("phrase create")
	if !ok {
		slog.Error("got message with unexpected type")
		os.Exit(1)
	}

	Phrase = msg.Phrase
	fmt.Println("Phrase generated for file transfer:", Phrase)

	msg, ok = ws.GetMessageWithExpectedType("answer")
	if !ok {
		slog.Error("got message with unexpected type")
		os.Exit(1)
	}

	answerSDP, err := msg.ToSessionDescription()
	if err != nil {
		slog.Error(err.Error())
	}

	if err := rtc.SetRemoteDescription(*answerSDP); err != nil {
		slog.Error("unable to set remote description", "error", err.Error())
	}

	// fmt.Println("running ice candidate in goroutine")
	go rtc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			ws.SendIceCandidate(candidate)
		}
	})
	// fmt.Println("ice candidate go routine running, waiting for ice candidate message")

	// fmt.Println("ice gathering state", rtc.ICEGatheringState())
	// fmt.Println("current local description", rtc.CurrentLocalDescription())
	// fmt.Println("current remote description", rtc.CurrentRemoteDescription())
	// fmt.Println()

	iceCandidateMsg, ok := ws.GetMessageWithExpectedType("ice candidate")
	if !ok {
		slog.Error("got message with unexpected type")
	}

	candidate, err := iceCandidateMsg.ToIceCandidate()
	if err != nil {
		slog.Error("error getting ice candidate", "error", err)
	}

	rtc.AddICECandidate(*candidate)

	for {
		fmt.Println(rtcDataChan.ReadyState())
		fmt.Println(rtc.PeerConnection.ICEConnectionState())
		time.Sleep(time.Second * 1)
	}

}

func collectOperations(flags *Flags) {
	Phrase = flags.CollectCode

	ws, err := WebsocketConnect(*flags.Server)
	if err != nil {
		slog.Error("unable to initialise websocket connection", "error", err.Error())
	}
	defer ws.Close()

	rtc, err := CreatePeerConnection()
	if err != nil {
		slog.Error("unable to create peer connection", "error", err.Error())
	}
	defer rtc.Close()

	// Log ICE connection state changes
	rtc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", state.String())
	})

	// Log signaling state changes
	rtc.OnSignalingStateChange(func(state webrtc.SignalingState) {
		fmt.Printf("Signaling State has changed: %s\n", state.String())
	})

	// Log connection state changes
	rtc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("PeerConnection State has changed: %s\n", state.String())
		if state == webrtc.PeerConnectionStateFailed {
			fmt.Println("Connection has failed. Additional debugging information may be needed.")
		}
	})

	// Log ICE candidate gathering state changes
	rtc.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		fmt.Printf("ICE Gathering State has changed: %s\n", state.String())
	})

	rtc.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		fmt.Println("connection state change", rtc.ConnectionState())
	})

	rtcDataChan, err := rtc.CreateDataChannel()
	if err != nil {
		slog.Error("unable to create data channel", "error", err.Error())
	}
	defer rtcDataChan.Close()

	rtc.HandleDataChannel()

	if err := ws.GetOffer(); err != nil {
		slog.Error("unable to get offer from sender", "error", err.Error())
	}

	msg, ok := ws.GetMessageWithExpectedType("offer")
	if !ok {
		slog.Error("got message with unexpected type")
		os.Exit(1)
	}

	offerSDP, err := msg.ToSessionDescription()
	if err != nil {
		slog.Error(err.Error())
	}

	if err := rtc.SetRemoteDescription(*offerSDP); err != nil {
		slog.Error("unable to set remote description", "error", err.Error())
	}

	answerSDP, err := rtc.CreateAnswer()
	if err != nil {
		slog.Error("unable to create answer", "error", err.Error())
	}

	if err := ws.SendWebrtcSessionDescription(answerSDP); err != nil {
		slog.Error("unable to send answer", "error", err.Error())
	}

	_, ok = ws.GetMessageWithExpectedType("success")
	if !ok {
		slog.Error("got message with unexpected type")
	}

	go rtc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			ws.SendIceCandidate(candidate)
		}
	})

	iceCandidateMsg, ok := ws.GetMessageWithExpectedType("ice candidate")
	if !ok {
		slog.Error("got message with unexpected type")
	}

	candidate, err := iceCandidateMsg.ToIceCandidate()
	if err != nil {
		slog.Error("error getting ice candidate", "error", err)
	}

	rtc.AddICECandidate(*candidate)

	for {
		fmt.Println(rtcDataChan.ReadyState())
		fmt.Println(rtc.PeerConnection.ICEConnectionState())
		time.Sleep(time.Second * 1)
	}

	time.Sleep(time.Second * 5)

	sendMessage := func(message string) {
		err := rtcDataChan.SendText(message)
		if err != nil {
			fmt.Println("Error sending message:", err)
		}
	}

	sendMessage("webrtc test message")

}
