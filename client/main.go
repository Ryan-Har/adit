package main

import (
	"fmt"
	"github.com/pion/webrtc/v3"
	"log/slog"
	"os"
)

type action string

const (
	Sender    action = "sender"
	Collector action = "collector"
)

func main() {
	flags, err := GetFlags()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: flags.logLevel})))

	establishConnection(flags)
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

	rtcDataChan, err := rtc.CreateDataChannel()
	if err != nil {
		slog.Error("unable to create data channel", "error", err.Error())
	}
	defer rtcDataChan.Close()

	go ws.HandleIncomingMessages(rtc)

	rtc.LogChanges()
	rtc.HandleDataChannel()

	switch runType {
	case Sender:
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

	select {}

}
