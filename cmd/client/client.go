package main

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
)

var (
	server   string
	psk      string
	interval time.Duration
	insecure bool
	version  string = "dev"
	commit   string = "none"
	date     string = "unknown"
)

var banner = `PeroniBOT [/root presence bot] - client -`

func main() {
	fmt.Printf("%s %v, commit %v, built at %v\n", banner, version, commit, date)

	flag.StringVar(&server, "server", "localhost:8081", "server address")
	flag.StringVar(&psk, "psk", "", "pre-shared key")
	flag.DurationVar(&interval, "interval", 15*time.Second, "timeout interval")
	flag.BoolVar(&insecure, "insecure", false, "ok for insecure")

	flag.Parse()

	var pingMessage = fmt.Sprintf("ping-%s", psk)
	var pongMessage = fmt.Sprintf("pong-%s", psk)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	url := url.URL{
		Scheme: "wss",
		Host:   server,
		Path:   "/rootbot",
	}
	log.Printf("Making connection to: %s", url.String())

	if insecure {
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	conn, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to dial: %s. %s\n", url.String(), err)
		log.Printf("Failed to dial: %s. %s", url.String(), err)
		os.Exit(1)
	}
	defer conn.Close()

	done := make(chan struct{})
	go func() {
		defer conn.Close()
		defer close(done)

		for {
			log.Println("Sending: ping.")
			err = conn.WriteMessage(websocket.TextMessage, []byte(pingMessage))
			if err != nil {
				log.Println("Write Error: ", err)
				break
			}

			msgType, bytes, err := conn.ReadMessage()
			if err != nil {
				log.Println("WebSocket closed.")
				return
			}
			if msg := string(bytes[:]); msgType != websocket.TextMessage && msg != pongMessage {
				log.Println("Unrecognized message received.")
				continue
			} else {
				log.Println("Received: pong.")
			}

			time.Sleep(interval)
		}
	}()

	for {
		select {
		case <-interrupt:
			log.Println("Client interrupted.")
			err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("WebSocket Close Error: ", err)
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		case <-done:
			log.Println("WebSocket connection terminated.")
			return
		}
	}
}
