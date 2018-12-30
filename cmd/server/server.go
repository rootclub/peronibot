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
	TLSCert    string
	TLSCertKey string
	interval   time.Duration
	isOpen     bool   = false
	version    string = "dev"
	commit     string = "none"
	date       string = "unknown"
)

var banner = `PeroniBOT [/root presence bot] - server -`

func main() {
	fmt.Printf("%s %v, commit %v, built at %v\n", banner, version, commit, date)

	flag.StringVar(&port, "port", ":8081", "server port")
	flag.StringVar(&psk, "psk", "", "pre-shared key")
	flag.StringVar(&tgtoken, "tgtoken", "", "telegram token")
	flag.StringVar(&TLSCert, "tlscert", "cert.pem", "TLS Certificate file")
	flag.StringVar(&TLSCertKey, "tlscertkey", "key.pem", "TLS Certificate key")
	flag.Int64Var(&tgGroupID, "tggroupid", 0, "telegram group id")
	flag.DurationVar(&interval, "interval", 15*time.Second, "timeout interval")

	flag.Parse()
	http.HandleFunc("/rootbot", handler)
	http.ListenAndServeTLS(port, TLSCert, TLSCertKey, nil)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func handler(res http.ResponseWriter, req *http.Request) {

	bot, err := tgbotapi.NewBotAPI(tgtoken)
	if err != nil {
		log.Fatal("Cannot log in, exiting...")
	}
	botUsername := fmt.Sprintf("@%s", bot.Self.UserName)
	log.Infof("Authorized on account %s", botUsername)

	var pongMessage = fmt.Sprintf("pong-%s", psk)
	var pingMessage = fmt.Sprintf("ping-%s", psk)
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Errorf("Upgrade Error: ", err)
		return
	}

	defer conn.Close()

	for {
		msgType, bytes, err := conn.ReadMessage()
		if err != nil {
			log.Warnf("Read Error: ", err)
			break
		}
		if msg := string(bytes[:]); msgType != websocket.TextMessage || msg != pingMessage {
			log.Warnf("unknown message %q\n", msg)
			break
		}

		log.Debugf("Received: ping.")

		if !isOpen {
			log.Infof("Il ROOT è aperto!")
			msg := tgbotapi.NewMessage(tgGroupID, "Il ROOT è aperto!")
			bot.Send(msg)
			isOpen = true
		}

		time.Sleep(interval)
		log.Debugf("Sending: pong.")
		err = conn.WriteMessage(websocket.TextMessage, []byte(pongMessage))
		if err != nil {
			log.Println("Write Error: ", err)
			break
		}
	}

	if isOpen {
		log.Infof("Il ROOT è chiuso")
		msg := tgbotapi.NewMessage(tgGroupID, "Il ROOT è chiuso")
		bot.Send(msg)
		isOpen = false
	}
}
