package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/gorilla/websocket"

	"log/slog"
	"net/http"
	"os"
	//"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Add validation logic here if needed
		return true
	},
}

// TODO: Handle timeouts of sessions so that this doesn't grow forever
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

func GetNumberOfWords(num int) (string, error) {
	data, err := os.ReadFile("wordlist.json")
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

	//ticker := time.NewTicker(time.Second)

	// for {
	// 	<-ticker.C
	// 	for _, v := range ongoingSessions {
	// 		fmt.Println(v)
	// 	}
	// }
}
