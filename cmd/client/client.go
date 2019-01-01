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
	version  = "dev"
	commit   = "none"
	date     = "unknown"
)

var banner = `PeroniBOT [/root presence bot] - client -`

func main() {
	log.Infof("%s %v, commit %v, built at %v\n", banner, version, commit, date)

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
	log.Infof("Making connection to: %s", url.String())

	if insecure {
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec
	}

	conn, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		log.Fatalf("failed to dial: %s. %s", url.String(), err)
	}
	defer func() {
		if err = conn.Close(); err != nil {
			log.Errorf("error during connection close: %s", err)
		}
	}()

	done := make(chan struct{})
	go func() {

		defer func() {
			if err = conn.Close(); err != nil {
				log.Errorf("error during connection close: %s", err)
			}
			close(done)
		}()

		for {
			log.Debug("sending: ping.")
			if err = conn.WriteMessage(websocket.TextMessage, []byte(pingMessage)); err != nil {
				log.Errorf("write error: %s", err)
				break
			}

			var msgType int
			var bytes []byte
			msgType, bytes, err = conn.ReadMessage()
			if err != nil {
				log.Info("websocket closed.")
				return
			}
			if msg := string(bytes[:]); msgType != websocket.TextMessage && msg != pongMessage {
				log.Warn("unrecognized message received.")
				continue
			} else {
				log.Debug("received: pong.")
			}

			time.Sleep(interval)
		}
	}()

	for {
		select {
		case <-interrupt:
			log.Debug("client interrupted.")
			err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Errorf("websocket close error: %s", err)
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		case <-done:
			log.Info("websocket connection terminated.")
			return
		}
	}
}
