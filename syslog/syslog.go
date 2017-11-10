package syslog

import (
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/hpcloud/tail"
)

const timeFormat = "Jan 2 15:04:05"

var reply = regexp.MustCompile("^(.+) [a-z]+ dnsmasq.+: reply ([^ ]+) is ([^ ]+)")
var query = regexp.MustCompile("^(.+) [a-z]+ dnsmasq.+: query.A. ([^ ]+) from ([^ ]+)")
var ack = regexp.MustCompile("^(.+) [a-z]+ dnsmasq-dhcp.+: DHCPACK.+ ([^ ]+) ([^ ]+) ([^ ]+)")

// Device : A representation of a DHCP request
type Device struct {
	At       *time.Time
	Hostname string
	Mac      string
	IP       string
}

// Request : A representation of a DNS request
type Request struct {
	At      *time.Time
	Host    string
	Source  string
	Aliases map[string]string
}

// Tail will tail the log and send a device / request
// to the appropriate channel when it is found
func Tail(path string) (chan *Device, chan *Request, error) {
	devices := make(chan *Device, 3)
	requests := make(chan *Request, 10)
	t, err := tail.TailFile(path, tail.Config{ReOpen: true, Follow: true})
	if err != nil {
		return nil, nil, err
	}
	go processFile(t, devices, requests)
	return devices, requests, nil
}

func setupExitListener(t *tail.Tail) {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT)
	go func() {
		<-signals
		t.Stop()
	}()
}

func parseQuery(requests chan *Request, match *[]string) *Request {
	at, _ := time.Parse(timeFormat, (*match)[1])
	latest := &Request{&at, (*match)[2], (*match)[3], map[string]string{}}
	log.Printf("Found request: %v\n", latest)
	requests <- latest
	return latest
}

func parseReply(latest *Request, match *[]string) {
	latest.Aliases[(*match)[3]] = (*match)[2]
}

func parseAck(devices chan *Device, match *[]string) {
	at, _ := time.Parse(timeFormat, (*match)[1])
	device := &Device{&at, (*match)[4], (*match)[3], (*match)[2]}
	log.Printf("Found device: %v\n", device)
	devices <- device
}

func processFile(t *tail.Tail, devices chan *Device, requests chan *Request) {
	var current *Request
	count := 0
	for line := range t.Lines {
		if count%100 == 0 {
			log.Printf("%d lines read\n", count)
		}
		count++
		if match := query.FindStringSubmatch(line.Text); match != nil {
			current = parseQuery(requests, &match)
		} else if match = reply.FindStringSubmatch(line.Text); match != nil {
			parseReply(current, &match)
		} else if match = ack.FindStringSubmatch(line.Text); match != nil {
			parseAck(devices, &match)
		}
	}
}
