package main

import (
	"fmt"
	"log/slog"
	"os"
	//"time"

	"github.com/pion/webrtc/v3"
)

type action string

const (
	Sender    action = "sender"
	Collector action = "collector"
)

func init() {
	SerialisedChunks = make(map[int]FilePacket)
}

func main() {
	flags, err := GetFlags()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: flags.logLevel})))

	establishConnection(flags)
	select {}
}

func establishConnection(flags *Flags) {
	var runType action

	//TODO: Validate file / folder before making expensive network call
	if flags.InputFile != "" {
		runType = Sender
	}
	if flags.CollectCode != "" {
		runType = Collector
	}

	if runType == Sender {
		if _, err := readFileInChunks(flags.InputFile, flags.ChunkSize); err != nil {
			slog.Error("unable to read provided file", "error", err.Error())
			os.Exit(1)
		}
	}

	ws, err := WebsocketConnect(*flags.Server)
	if err != nil {
		slog.Error("unable to initialise websocket connection", "error", err.Error())
	}
	//defer ws.Close()

	rtc, err := CreatePeerConnection()
	if err != nil {
		slog.Error("unable to create peer connection", "error", err.Error())
	}
	//defer rtc.Close()

	rtcDataChan, err := rtc.CreateDataChannel(runType, flags)
	if err != nil {
		slog.Error("unable to create data channel", "error", err.Error())
	}
	//defer rtcDataChan.Close()
	//rtc.HandleDataChannel(runType, flags)
	go ws.HandleIncomingMessages(rtc)

	rtc.HandleChanges(ws)
	//HandleDataChannel(rtcDataChan, runType, flags)

	//rtc.HandleDataChannel(runType, flags)

	switch runType {
	case Sender:
		rtc.HandleRetransmission(rtcDataChan, flags)
		offerSDP, err := rtc.CreateOffer()
		if err != nil {
			slog.Error("unable to create offer", "error", err.Error())
		}

		if err := ws.SendWebrtcSessionDescription(offerSDP); err != nil {
			slog.Error("unable to send offer", "error", err.Error())
		}

		//wait for answer to return
		for {
			if ws.answerSDP.SDP != "" {
				break
			}
		}

	case Collector:
		ws.Phrase = flags.CollectCode
		rtc.HandleFileReception(rtcDataChan, flags)
		// handleFileReception(rtcDataChan, rtc, flags)
		if err := ws.GetOffer(); err != nil {
			slog.Error("unable to get offer from sender", "error", err.Error())
		}

		//wait for offerSDP to return
		for {
			if ws.offerSDP.SDP != "" {
				break
			}
		}

		answerSDP, err := rtc.CreateAnswer()
		if err != nil {
			slog.Error("unable to create answer", "error", err.Error())
		}

		if err := ws.SendWebrtcSessionDescription(answerSDP); err != nil {
			slog.Error("unable to send answer", "error", err.Error())
		}
	}

	rtc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			ws.SendIceCandidate(candidate)
		}
	})

}
