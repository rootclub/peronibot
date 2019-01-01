package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var (
	port       string
	psk        string
	tgtoken    string
	tgGroupID  int64
	tlsCert    string
	tlsCertKey string
	interval   time.Duration
	isOpen     = false
	version    = "dev"
	commit     = "none"
	date       = "unknown"
)

const (
	openMessage  string = "Il ROOT è aperto!"
	closeMessage string = "Il ROOT è chiuso"
)

var banner = `PeroniBOT [/root presence bot] - server -`

func main() {
	log.Infof("%s %v, commit %v, built at %v\n", banner, version, commit, date)

	flag.StringVar(&port, "port", ":8081", "server port")
	flag.StringVar(&psk, "psk", "", "pre-shared key")
	flag.StringVar(&tgtoken, "tgtoken", "", "telegram token")
	flag.StringVar(&tlsCert, "tlscert", "cert.pem", "TLS Certificate file")
	flag.StringVar(&tlsCertKey, "tlscertkey", "key.pem", "TLS Certificate key")
	flag.Int64Var(&tgGroupID, "tggroupid", 0, "telegram group id")
	flag.DurationVar(&interval, "interval", 15*time.Second, "timeout interval")

	flag.Parse()
	http.HandleFunc("/rootbot", handler)
	if err := http.ListenAndServeTLS(port, tlsCert, tlsCertKey, nil); err != nil {
		log.Fatalf("can't start https listener: %s", err)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func handler(res http.ResponseWriter, req *http.Request) {

	bot, err := tgbotapi.NewBotAPI(tgtoken)
	if err != nil {
		log.Fatal("cannot log in, exiting...")
	}
	botUsername := fmt.Sprintf("@%s", bot.Self.UserName)
	log.Infof("authorized on account %s", botUsername)

	var pongMessage = fmt.Sprintf("pong-%s", psk)
	var pingMessage = fmt.Sprintf("ping-%s", psk)
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Errorf("upgrade error: %s", err)
		return
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("error during connection close: %s", err)
		}
	}()

	for {
		msgType, bytes, err := conn.ReadMessage()
		if err != nil {
			log.Warnf("read error: %s", err)
			break
		}
		if msg := string(bytes[:]); msgType != websocket.TextMessage || msg != pingMessage {
			log.Warnf("unknown message %q\n", msg)
			break
		}

		log.Debugf("received: ping.")

		if !isOpen {
			log.Info(openMessage)
			msg := tgbotapi.NewMessage(tgGroupID, openMessage)
			if _, err = bot.Send(msg); err != nil {
				log.Errorf("cannot send 'open' message to telegram: %s", err)
			}
			isOpen = true
		}

		time.Sleep(interval)
		log.Debugf("sending: pong.")
		err = conn.WriteMessage(websocket.TextMessage, []byte(pongMessage))
		if err != nil {
			log.Errorf("write Error: %s", err)
			break
		}
	}

	if isOpen {
		log.Info(closeMessage)
		msg := tgbotapi.NewMessage(tgGroupID, closeMessage)
		if _, err := bot.Send(msg); err != nil {
			log.Errorf("cannot send 'close' message to telegram: %s", err)
		}
		isOpen = false
	}
}
