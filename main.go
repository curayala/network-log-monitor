package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jasonlvhit/gocron"
	"github.com/tmullender/network-log-monitor/notify"
	"github.com/tmullender/network-log-monitor/state"
	"github.com/tmullender/network-log-monitor/syslog"
	"github.com/tmullender/network-log-monitor/ui"
)

// Config : The configuration needed
type Config struct {
	DbURL        string
	HTTPHost     string
	HTTPAddress  string
	LogPath      string
	MailInterval uint64
	MailConfig   *notify.Config
}

func main() {
	config := readConfig()
	store, err := state.NewStore(config.DbURL)
	defer store.Close()
	exitOnError(err)
	startProcessing(config.LogPath, store)
	startUserInterface(config, store)
	startScheduler(config, store)
}

func readConfig() *Config {
	cfg := flag.String("cfg", "", "The configuration file to use")
	flag.Parse()
	config := &Config{"network-log.db", ":8080", "http://localhost:8080", flag.Arg(0), 0, nil}
	if len(*cfg) > 0 {
		file, err := os.Open(*cfg)
		exitOnError(err)
		err = json.NewDecoder(file).Decode(config)
		exitOnError(err)
	}
	return config
}

func startProcessing(path string, store *state.Store) {
	devices, requests, err := syslog.Tail(path)
	exitOnError(err)
	log.Println("Starting file processing")
	go process(devices, requests, store)
}

func startUserInterface(config *Config, store *state.Store) {
	server := &http.Server{Addr: config.HTTPHost}
	setupExitListener(server)
	go startServer(server, store, config.HTTPAddress)
}

func startScheduler(config *Config, store *state.Store) {
	if config.MailInterval > 0 {
		log.Println("Starting cron")
		gocron.Every(config.MailInterval).Minutes().Do(sendUpdate, config, store)
	}
	<-gocron.Start()
}

func process(devices chan *syslog.Device, requests chan *syslog.Request, store *state.Store) {
	for true {
		select {
		case device := <-devices:
			store.AddDevice(device.At, device.Hostname, device.IP, device.Mac)
		case request := <-requests:
			if _, authorized := (*store.GetAuthorisedHosts())[request.Host]; !authorized {
				handleRequest(request, store)
			}
		}
	}
}

func handleRequest(request *syslog.Request, store *state.Store) {
	device := store.FindDeviceByIP(request.Source)
	log.Printf("handleRequest %v for %v\n", request, device)
	if device == nil {
		device = store.AddDevice(&time.Time{}, request.Source, request.Source, request.Source)
	}
	if _, ignored := (*store.GetIgnoredDevices())[device.Mac]; ignored {
		return
	}
	device.AddRequest(request.At, request.Host)
}

func startServer(server *http.Server, store *state.Store, address string) {
	log.Println("Starting UI")
	http.HandleFunc("/", ui.Root())
	http.HandleFunc("/authorized-hosts", ui.GetAuthorizedHosts(store))
	http.HandleFunc("/authorized-hosts/add", ui.AddAuthorizedHosts(store))
	http.HandleFunc("/authorized-hosts/remove", ui.RemoveAuthorizedHosts(store))
	http.HandleFunc("/ignored-devices", ui.GetIgnoredDevices(store))
	http.HandleFunc("/ignored-devices/add", ui.AddIgnoredDevice(store))
	http.HandleFunc("/ignored-devices/remove", ui.RemoveIgnoredDevice(store))
	http.HandleFunc("/latest", ui.Latest(store, address))
	err := server.ListenAndServe()
	exitOnError(err)
}

func setupExitListener(server *http.Server) {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT)
	go func() {
		<-signals
		server.Shutdown(nil)
		gocron.Clear()
	}()
}

func exitOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func sendUpdate(config *Config, store *state.Store) {
	log.Println("Sending update")
	err := notify.SendUpdate(config.MailConfig, &notify.Content{
		Devices: store.GetLatestRequests(true),
		Root:    config.HTTPAddress,
	})
	if err != nil {
		log.Println(err)
	}
}
