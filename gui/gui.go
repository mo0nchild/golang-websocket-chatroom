package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	"github.com/gorilla/websocket"
)

var (
	clientConnected bool = false
	wsConnection    *websocket.Conn
	urlArdess       url.URL
	userName        string
)

type userData struct {
	url      string
	UserName string `json:"username"`
	Message  string `json:"message"`
	Time     string `json:"time"`
}

type serverDataBuffer struct {
	Data []userData `json:"data"`
}

func (ud userData) sendDataToServer(conn *websocket.Conn) error {
	jsonData, _ := json.Marshal(ud)
	if err := conn.WriteJSON(jsonData); err != nil {
		return err
	}
	return nil
}

func updateWebSocket(w *astilectron.Window, done chan struct{},
	serverMsg chan serverDataBuffer) {
	for {
		select {
		case <-done:
			clientConnected = false
			w.SendMessage("disconnect")
			fmt.Println("close")

		case message := <-serverMsg:

			log.Println(message.Data)
			var textarea string
			for _, msg := range message.Data {
				textarea += fmt.Sprintf("%s|%s:%s\n", msg.Time, msg.UserName, msg.Message)
			}
			w.SendMessage(textarea)
		}
	}
}

func getServerMSG(conn *websocket.Conn, serverMsg chan serverDataBuffer,
	done chan struct{}) {
	var (
		serverData serverDataBuffer
		byteData   []byte
	)
	defer conn.Close()
	for {
		select {
		case <-done:
			return
		default:
			err := conn.ReadJSON(&byteData)
			if err != nil {
				log.Println("read:", err)
				done <- struct{}{}
				return
			}
			json.Unmarshal(byteData, &serverData)
			log.Println(serverData)
			serverMsg <- serverData
		}
	}
}

func getUserData(d string) userData {

	var data map[string]string
	for i, chr := range d[2:len(d)] {
		if string(chr) == "|" {
			data = map[string]string{
				"url":      d[2 : i+2],
				"username": d[i+3 : len(d)],
			}
			break
		}
	}

	log.Printf("Username:%s, Adress:%s", data["username"], data["url"])
	url := url.URL{Scheme: "wss", Host: data["url"], Path: "/ws"}
	log.Printf("connecting to %s", url.String())

	return userData{
		url:      url.String(),
		UserName: data["username"],
	}
}

func main() {

	a, err := astilectron.New(nil, astilectron.Options{
		AppName:           "ChatClient",
		BaseDirectoryPath: "gui",
	})
	if err != nil {
		log.Fatal(fmt.Errorf("main: creating astilectron failed: %w", err))
	}
	defer a.Close()

	// Handle signals
	a.HandleSignals()

	// Start
	if err = a.Start(); err != nil {
		log.Fatal(fmt.Errorf("main: starting astilectron failed: %w", err))
	}

	// New window
	var w *astilectron.Window
	if w, err = a.NewWindow("./gui.html", &astilectron.WindowOptions{
		Center:    astikit.BoolPtr(true),
		Height:    astikit.IntPtr(550),
		Width:     astikit.IntPtr(700),
		MinHeight: astikit.IntPtr(550),
		MinWidth:  astikit.IntPtr(700),
		MaxHeight: astikit.IntPtr(550),
		MaxWidth:  astikit.IntPtr(700),
	}); err != nil {
		log.Fatal(fmt.Errorf("main: new window failed: %w", err))
	}

	updateChecking := make(chan struct{})
	wsMessage := make(chan serverDataBuffer)
	var data userData
	go updateWebSocket(w, updateChecking, wsMessage)

	// Create windows
	if err = w.Create(); err != nil {
		log.Fatal(fmt.Errorf("main: creating window failed: %w", err))
	}

	w.On(astilectron.EventNameAppClose, func(e astilectron.Event) (deleteListener bool) {
		wsConnection.Close()
		return
	})

	w.OnMessage(func(m *astilectron.EventMessage) interface{} {

		var requestString string
		m.Unmarshal(&requestString)

		switch requestString[0:2] {
		case "B:":
			if clientConnected {
				data.Time = time.Now().Format("2006.01.02 15:04:05")
				data.Message = requestString[2:len(requestString)]
				data.sendDataToServer(wsConnection)
			}
		case "C:":
			if !clientConnected {
				data = getUserData(requestString)
				wsConnection, _, err = websocket.DefaultDialer.Dial(data.url, nil)
				if err != nil {
					log.Println(err)
					return "false"
				}
				go getServerMSG(wsConnection, wsMessage, updateChecking)
			} else {
				updateChecking <- struct{}{}
			}
			clientConnected = !clientConnected
			return strconv.FormatBool(clientConnected)
		}
		return nil
	})

	//w.OpenDevTools()
	a.Wait()

}
