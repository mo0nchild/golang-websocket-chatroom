package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type requestData struct {
	UserName string `json:"username"`
	Message  string `json:"message"`
	Time     string `json:"time"`
}

type userDataBuffer struct {
	Data []requestData `json:"data"`
}

func (udb userDataBuffer) sendDataToClient(conn *websocket.Conn) error {
	jsonData, _ := json.Marshal(udb)
	if err := conn.WriteJSON(jsonData); err != nil {
		return err
	}
	return nil
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

var (
	dataBuffer userDataBuffer
	byteData   []byte
	data       requestData
)

func updateRequest(conn *websocket.Conn) {
	var count int
	go func() {
		for {
			if count != len(dataBuffer.Data) {
				dataBuffer.sendDataToClient(conn)
				count++
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()
	for {
		err := conn.ReadJSON(&byteData)
		if err != nil {
			log.Println(err)
			return
		}

		json.Unmarshal(byteData, &data)
		dataBuffer.Data = append(dataBuffer.Data, data)

		file, _ := json.MarshalIndent(dataBuffer, "", " ")
		ioutil.WriteFile("data.json", file, 0600)
		fmt.Printf("Username: %s,\t Message: %s\n", data.UserName, data.Message)

	}
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("Client Connected")

	updateRequest(ws)
}

func determineListenAddress() (string, error) {
	port := os.Getenv("PORT")
	if port == "" {
		return "", fmt.Errorf("$PORT not set")
	}
	return ":" + port, nil
}

func searchIPAddress() {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Println(err)
		return
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				log.Println(ipnet.IP.String() + "\n")
			}
		}
	}
}

func main() {
	addr, err := determineListenAddress()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(addr)
	searchIPAddress()

	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndpoint)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
