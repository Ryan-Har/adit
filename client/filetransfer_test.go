package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshallMetadata(t *testing.T) {
	metadata := FileMetadata{
		FileName:  "testfile.txt",
		FileSize:  12345,
		NumChunks: 10,
	}

	jsonData, err := json.Marshal(metadata)
	assert.NoError(t, err)

	// Test the unmarshal function
	result, err := unmarshallMetadata(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, metadata, result)
}

func TestUnmarshallFilePacket(t *testing.T) {
	packet := FilePacket{
		SequenceNumber: 1,
		Data:           []byte("chunk data"),
	}

	jsonData, err := json.Marshal(packet)
	assert.NoError(t, err)

	result, err := unmarshallFilePacket(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, packet, result)
}

// Test checkForMissingChunks function
func TestCheckForMissingChunks(t *testing.T) {
	SerialisedChunks = make(map[int]FilePacket)
	SerialisedChunks[0] = FilePacket{SequenceNumber: 0, Data: []byte("chunk1")}
	SerialisedChunks[2] = FilePacket{SequenceNumber: 2, Data: []byte("chunk3")}

	// Case with missing chunks
	missing, complete := checkForMissingChunks(3)
	assert.Equal(t, []int{1}, missing)
	assert.False(t, complete)

	// Case with no missing chunks
	SerialisedChunks[1] = FilePacket{SequenceNumber: 1, Data: []byte("chunk2")}
	missing, complete = checkForMissingChunks(3)
	assert.Empty(t, missing)
	assert.True(t, complete)
}
