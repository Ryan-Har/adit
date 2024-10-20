package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/gorilla/websocket"

	"log/slog"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Add validation logic here if needed
		return true
	},
}

var ongoingSessions = make(map[string]*Peers)

func wsUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade error", "error", err)
		return
	}
	slog.Info("new websocket connection", "remoteAddr", conn.RemoteAddr())
	//don't close here, handleConnection will close
	p := Peer{
		Conn: conn,
	}

	go p.handleConnection()
}

//go:embed wordlist.json
var wordlistFile embed.FS

func GetNumberOfWords(num int) (string, error) {
	data, err := wordlistFile.ReadFile("wordlist.json")
	if err != nil {
		return "", fmt.Errorf("error readint wordlist file %v", err.Error())
	}

	var words []string
	err = json.Unmarshal(data, &words)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling words to slice %v", err.Error())
	}

	var chosenWords []string
	for range num {
		chosenWords = append(chosenWords, words[rand.Intn(len(words)-1)])
	}

	return strings.Join(chosenWords, "."), nil

}

func main() {
	http.HandleFunc("/ws", wsUpgrade)
	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
