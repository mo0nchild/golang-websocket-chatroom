package main

import (
	"flag"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {

	flag.Parse()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	done := make(chan struct{})
	msg := make(chan []byte)

	go func() {
		defer close(done)
		// var _message string

		messageType, m, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(string(m))

		for {
			// fmt.Scanf("%s", _message)
			conn.WriteMessage(messageType, []byte("User Messange"))
			log.Println(messageType)
			msg <- []byte("")
			time.Sleep(2 * time.Second)
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-msg:
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Println(string(message))
		}
	}
}
