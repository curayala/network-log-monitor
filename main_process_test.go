package main

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/tmullender/network-log-monitor/state"
	"github.com/tmullender/network-log-monitor/syslog"
)

const deviceCount = 10

func TestProcess(t *testing.T) {
	devices := make(chan *syslog.Device)
	requests := make(chan *syslog.Request)
	go createEvents("proc", devices, requests)

	store, _ := state.NewStore("/tmp/processing")
	defer store.Close()
	defer os.Remove("/tmp/processing")
	go process(devices, requests, store)

	validate(t, store, deviceCount)
}

func TestAuthorizedHost(t *testing.T) {
	devices := make(chan *syslog.Device)
	requests := make(chan *syslog.Request)
	go createEvents("auth", devices, requests)

	store, _ := state.NewStore("/tmp/authorized")
	defer store.Close()
	defer os.Remove("/tmp/authorized")
	store.AuthoriseHost("www.auth0.com")
	store.IgnoreDevice("AA:BB:CC:DD:EE:F3")
	go process(devices, requests, store)

	validate(t, store, deviceCount-1)
}

func TestUnknownDevice(t *testing.T) {
	devices := make(chan *syslog.Device)
	requests := make(chan *syslog.Request)
	go func() {
		requests <- &syslog.Request{
			At:      &time.Time{},
			Host:    "www.google.com",
			Source:  "127.0.0.1",
			Aliases: map[string]string{},
		}
	}()

	store, _ := state.NewStore("/tmp/unknown")
	defer store.Close()
	defer os.Remove("/tmp/unknown")
	go process(devices, requests, store)

	time.Sleep(time.Second)
	latest := store.GetLatestRequests(false)
	for device, hosts := range *latest {
		if device.Mac != "127.0.0.1" || len(*hosts) != 1 {
			log.Printf("Failing %v with %v\n\n", device, hosts)
			t.Fail()
		}
	}
}

func createEvents(prefix string, devices chan *syslog.Device, requests chan *syslog.Request) {
	at := time.Now()
	for i := 0; i < deviceCount; i++ {
		at = at.Add(time.Second)
		devices <- &syslog.Device{
			At:       &at,
			Hostname: fmt.Sprintf("%s%d", prefix, i),
			IP:       fmt.Sprintf("127.0.0.%d", i),
			Mac:      fmt.Sprintf("AA:BB:CC:DD:EE:F%d", i),
		}
		for j := 0; j < deviceCount; j++ {
			at = at.Add(time.Second)
			requests <- &syslog.Request{
				At:      &at,
				Host:    fmt.Sprintf("www.%s%d.com", prefix, j),
				Source:  fmt.Sprintf("127.0.0.%d", i),
				Aliases: map[string]string{},
			}
		}
	}
}

func validate(t *testing.T, store *state.Store, count int) {
	time.Sleep(time.Second)

	latest := *store.GetLatestRequests(true)
	if len(latest) != count {
		log.Printf("Expected %d devices, found %d\n\n", count, len(latest))
		t.Fail()
		return
	}
	for device, hosts := range latest {
		if len(*hosts) != count && len(*hosts) != 0 {
			log.Printf("Device %s has %v\n\n", device.Name(), hosts)
			t.Fail()
		}
	}
}
