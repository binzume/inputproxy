package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	port           = flag.Int("port", 9000, "http listen port")
	host           = flag.String("interface", "localhost", "http listen interface")
	secret         = flag.String("secret", "", "secret string (default: random)")
	enableKeyinput = flag.Bool("keyinput", false, "Enable keyinput")
	debugLog       = flag.Bool("debug", false, "Enable debugLog")
	defaultDisplay = flag.Int("defaultDisplay", 0, "default display id (windows)")
)

func genRandom(n int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_" // len:64
	b := make([]byte, n)
	r := make([]byte, n)
	if _, err := rand.Read(r); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = letters[r[i]%byte(len(letters))]
	}
	return string(b)
}

var WSUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Target struct {
	Type string `json:"type"`
	ID   uint64 `json:"id"`
}

type InputEvent struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Target Target `json:"target"` // monitor | window

	// Mouse
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Button int     `json:"button"`

	// Key
	Key string `json:"key"`
}

func handleMouse(msg *InputEvent) {
	if msg.Target.Type == "window" {
		if !moveW(msg.X, msg.Y, msg.Target.ID) {
			return
		}
	} else if msg.Target.Type == "monitor" {
		if !moveD(msg.X, msg.Y, msg.Target.ID) {
			return
		}
	}
	if msg.Action == "click" {
		click(msg.Button)
	} else if msg.Action == "down" || msg.Action == "mousedown" {
		buttonState(msg.Button, true)
	} else if msg.Action == "up" || msg.Action == "mouseup" {
		buttonState(msg.Button, false)
	}
}

func handkeKey(msg *InputEvent) {
	if msg.Action == "press" {
		key(msg.Key, 0)
	} else if msg.Action == "down" {
		keyState(msg.Key, true)
	} else if msg.Action == "up" {
		keyState(msg.Key, false)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	conn, err := WSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	defer conn.Close()

	for {
		var msg InputEvent
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			conn.WriteJSON(map[string]interface{}{"type": "error"})
			break
		}
		if *debugLog {
			log.Println(msg)
		}

		switch msg.Type {
		case "mouse":
			handleMouse(&msg)
			break
		case "key":
			if *enableKeyinput {
				handkeKey(&msg)
			}
			break
		}
	}
}

func main() {
	flag.Parse()
	name := *secret
	if name == "" {
		name = genRandom(32)
	}

	log.Printf("input socket url: ws://%s:%d/socket/%s \n", *host, *port, name)
	http.HandleFunc("/socket/"+name, handler)
	http.ListenAndServe(fmt.Sprintf("%s:%d", *host, *port), nil)
}
