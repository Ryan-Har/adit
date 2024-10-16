package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"github.com/pion/webrtc/v3"
	//"time"
)

type FileMetadata struct {
	FileName  string `json:"fileName"`
	FileSize  int64  `json:"fileSize"`
	NumChunks int    `json:"numChunks"`
}

type FilePacket struct {
	SequenceNumber int    `json:"seq"`
	Data           []byte `json:"data"`
}

type MissingPacketRequest struct {
	MissingSequences []int `json:"missingSequences"`
}

var SerialisedChunks map[int]FilePacket

func unmarshallMetadata(msgBytes []byte) (FileMetadata, error) {
	var m FileMetadata
	if err := json.Unmarshal(msgBytes, &m); err != nil {
		return FileMetadata{}, err
	}
	return m, nil
}

func unmarshallFilePacket(msgBytes []byte) (FilePacket, error) {
	var f FilePacket
	if err := json.Unmarshal(msgBytes, &f); err != nil {
		return FilePacket{}, err
	}
	return f, nil
}

func unmashallMissingPacketRequest(msgBytes []byte) (MissingPacketRequest, error) {
	var mp MissingPacketRequest
	if err := json.Unmarshal(msgBytes, &mp); err != nil {
		return MissingPacketRequest{}, err
	}
	return mp, nil
}

func readFileInChunks(filePath string, chunkSize int) ([][]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var chunks [][]byte
	for {
		chunk := make([]byte, chunkSize)
		n, err := file.Read(chunk)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}
		chunks = append(chunks, chunk[:n])
	}

	return chunks, nil
}

func getFileMetadata(filePath string, chunkSize int) (FileMetadata, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return FileMetadata{}, err
	}

	fileSize := fileInfo.Size()
	numChunks := int((fileSize + int64(chunkSize) - 1) / int64(chunkSize)) // Calculate total chunks

	metadata := FileMetadata{
		FileName:  fileInfo.Name(),
		FileSize:  fileSize,
		NumChunks: numChunks,
	}

	return metadata, nil
}

func sendFileMetadata(d *webrtc.DataChannel, md FileMetadata) error {
	metadataBytes, err := json.Marshal(md)
	if err != nil {
		return err
	}

	err = d.Send(metadataBytes)
	if err != nil {
		return err
	}

	return nil
}

func sendChunksWithSequence(d *webrtc.DataChannel, filePath string, chunkSize int) error {

	chunks, err := readFileInChunks(filePath, chunkSize)
	if err != nil {
		return err
	}

	for i, chunk := range chunks {
		packet := FilePacket{
			SequenceNumber: i,
			Data:           chunk,
		}

		SerialisedChunks[i] = packet //used to make any retransmissions quicker

		packetBytes, err := json.Marshal(packet)
		if err != nil {
			return fmt.Errorf("error serializing packet %d: %v", i, err)
		}

		err = sendBytes(d, packetBytes)
		if err != nil {
			return fmt.Errorf("error sending packet %d: %v", i, err)
		}
	}
	return nil
}

func sequenceFile(filePath string, chunkSize int) error {
	chunks, err := readFileInChunks(filePath, chunkSize)
	if err != nil {
		return err
	}

	for i, chunk := range chunks {
		packet := FilePacket{
			SequenceNumber: i,
			Data:           chunk,
		}

		SerialisedChunks[i] = packet //used to make any retransmissions quicker
	}
	return nil
}

func sendBytes(d *webrtc.DataChannel, b []byte) error {
	return d.Send(b)
}

func handleFileSending(d *webrtc.DataChannel, flags *Flags) {
	fmt.Println("handling file sending")
	metadata, err := getFileMetadata(flags.InputFile, flags.ChunkSize)
	if err != nil {
		slog.Error(err.Error())
	}

	if err = sendFileMetadata(d, metadata); err != nil {
		slog.Error("error sending file metadata", "error", err.Error())
	}

	fp := path.Clean(flags.InputFile)
	err = sendChunksWithSequence(d, fp, flags.ChunkSize)
	if err != nil {
		slog.Error("error sending file", "error", err.Error(), "last chunk sent", len(SerialisedChunks))
	}

	if err = d.SendText("done"); err != nil {
		slog.Error("error sending done message", "error", err.Error())
	}
}

// TODO: provide some kind of update on the screen
func writeToFile(outputPath string, chunks map[int][]byte, totalChunks int) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	for i := 0; i < totalChunks; i++ {
		if chunk, ok := chunks[i]; ok {
			_, err = file.Write(chunk)
			if err != nil {
				return fmt.Errorf("error writing chunk %d: %v", i, err)
			}
		}
	}

	fmt.Println("File successfully received and written!")
	return nil
}

// returns a map of the sequence of missing chunks and true if there are no missing chunks
func checkForMissingChunks(receivedChunks map[int][]byte, totalChunks int) ([]int, bool) {
	missingSeq := []int{}
	for i := 0; i < totalChunks; i++ {
		if _, ok := receivedChunks[i]; !ok {
			missingSeq = append(missingSeq, i)
		}
	}

	if len(missingSeq) > 0 {
		return missingSeq, false
	}

	return missingSeq, true
}

func requestMissingChunks(d *webrtc.DataChannel, missingSequences []int) error {
	if len(missingSequences) > 0 {
		request := MissingPacketRequest{MissingSequences: missingSequences}
		requestBytes, err := json.Marshal(request)
		if err != nil {
			return err
		}
		err = d.Send(requestBytes)
		return err
	}
	return nil
}

func handleRetransmission(d *webrtc.DataChannel, flags *Flags) {
	fmt.Println("handling retransmissions")

	d.OnMessage(func(msg webrtc.DataChannelMessage) {
		var request MissingPacketRequest
		err := json.Unmarshal(msg.Data, &request)
		if err != nil {
			fmt.Println("Error parsing retransmission request:", err)
			return
		}

		for _, seq := range request.MissingSequences {
			packet, ok := SerialisedChunks[seq]
			if !ok {
				slog.Error("rerequest of a chunk that does not exist")
				return
			}

			packetBytes, _ := json.Marshal(packet)
			d.Send(packetBytes)
			slog.Info("retransmission request fulfilled", "seq", seq)
		}

	})
}

func ensureDirExists(dirPath string) (string, bool) {
	cleanedPath := filepath.Clean(dirPath)

	info, err := os.Stat(cleanedPath)
	if err == nil && info.IsDir() {
		return cleanedPath, true
	}

	return cleanedPath, false
}

func verifyDir(dirPath string) error {
	p, ok := ensureDirExists(dirPath)
	if ok {
		return nil
	}
	return os.Mkdir(p, os.ModePerm)
}
