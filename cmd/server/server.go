//go:generate go-bindata-assetfs -pkg $GOPACKAGE assets/

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/websocket"
	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var (
	port            string
	psk             string
	tgtoken         string
	tgGroupID       int64
	tlsCert         string
	tlsCertKey      string
	isOpenTimestamp int32
	interval        time.Duration
	isOpen          = false
	version         = "dev"
	commit          = "none"
	date            = "unknown"
)

const (
	openMessage     string  = "Il ROOT è aperto!"
	closeMessage    string  = "Il ROOT è chiuso"
	spaceAPIVersion string  = "0.13"
	spaceName       string  = "Root"
	spaceWWW        string  = "https://www.rootclub.it"
	spaceAddress    string  = "Via Santa Croce, 6669, 47032 San Pietro In Guardiano FC"
	spaceLat        float64 = 44.2195512
	spaceLon        float64 = 12.2095288
)

var banner = `PeroniBOT [/root presence bot] - server -`

type spaceAPI struct {
	API                 string   `json:"api"`
	Space               string   `json:"space"`
	Logo                string   `json:"logo"`
	URL                 string   `json:"url"`
	Location            location `json:"location"`
	Contact             contact  `json:"contact"`
	State               state    `json:"state"`
	IssueReportChannels []string `json:"issue_report_channels"`
	Projects            []string `json:"projects,omitempty"`
}

type location struct {
	Address string  `json:"address,omitempty"`
	Lon     float64 `json:"lon"`
	Lat     float64 `json:"lat"`
}

type contact struct {
	Email    string `json:"email,omitempty"`
	Irc      string `json:"irc,omitempty"`
	Ml       string `json:"ml,omitempty"`
	Facebook string `json:"facebook,facebook"`
	Twitter  string `json:"twitter,omitempty"`
}

type state struct {
	Icon       icon  `json:"icon"`
	Open       bool  `json:"open"`
	LastChange int32 `json:"lastchange"`
}

type icon struct {
	Open   string `json:"open"`
	Closed string `json:"closed"`
}

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

	http.Handle("/assets/", http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo, Prefix: ""}))
	http.HandleFunc("/rootbot", botHandler)
	http.HandleFunc("/spaceapi.json", spaceapiHandler)
	if err := http.ListenAndServeTLS(port, tlsCert, tlsCertKey, nil); err != nil {
		log.Fatalf("can't start https listener: %s", err)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func botHandler(res http.ResponseWriter, req *http.Request) {

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
			isOpenTimestamp = int32(time.Now().Unix())
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
		isOpenTimestamp = int32(time.Now().Unix())
	}
}

func spaceapiHandler(res http.ResponseWriter, req *http.Request) {
	spaceOut := spaceAPI{
		API:   spaceAPIVersion,
		Space: spaceName,
		Logo:  fmt.Sprintf("https://%s/assets/logo.jpg", req.Host),
		URL:   spaceWWW,
		Location: location{
			Address: spaceAddress,
			Lon:     spaceLon,
			Lat:     spaceLat,
		},
		Contact: contact{
			Email:    "infocom@rootclub.it",
			Facebook: "https://www.facebook.com/circolo.root",
			Twitter:  "https://twitter.com/CircoloRoot",
		},
		IssueReportChannels: []string{"email"},
		State: state{
			Icon: icon{
				Open:   fmt.Sprintf("https://%s/assets/open.jpg", req.Host),
				Closed: fmt.Sprintf("https://%s/assets/closed.jpg", req.Host),
			},
			Open:       isOpen,
			LastChange: isOpenTimestamp,
		},
		Projects: []string{"https://github.com/rootclub"},
	}
	// Log spaceapi request on stdout
	log.Infof("SpaceAPI request from: %q", req.RemoteAddr)
	if err := json.NewEncoder(res).Encode(spaceOut); err != nil {
		log.Errorf("can't process JSON for spaceapi request: %q", err)
	}
}
