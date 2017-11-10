package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tmullender/network-log-monitor/state"
)

func TestStartProcessing(t *testing.T) {
	path := "/tmp/processing.log"
	db := "/tmp/processing.db"
	store, _ := state.NewStore(db)
	startProcessing(path, store)
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	f.Write([]byte("May 24 12:00:00 something dnsmasq-dhcp[123]: DHCPACK(eth0) 192.168.0.0 00:11:22:33:44:55 host1\n"))
	f.Write([]byte("May 24 12:00:00 something dnsmasq-dhcp[123]: DHCPACK(eth0) 192.168.0.0 00:11:22:33:44:55 host1\n"))
	f.Write([]byte("May 24 12:00:01 something dnsmasq-dhcp[124]: DHCPACK(eth0) 192.168.0.1 00:11:22:33:44:66 host2\n"))
	f.Write([]byte("May 24 12:00:02 something dnsmasq-dhcp[125]: DHCPACK(eth0) 192.168.0.2 00:11:22:33:44:77 host3\n"))
	f.Write([]byte("May 24 12:00:03 something dnsmasq[126]: query[A] www.google.com from 192.168.0.0\n"))
	f.Write([]byte("May 24 12:00:04 something dnsmasq[127]: query[A] api.google.com from 192.168.0.0\n"))
	f.Write([]byte("May 24 12:00:05 something dnsmasq[128]: query[A] www.amazon.com from 192.168.0.1\n"))
	f.Write([]byte("May 24 12:00:06 something dnsmasq[129]: query[A] api.amazon.com from 192.168.0.1\n"))
	f.Write([]byte("May 24 12:00:07 something dnsmasq[130]: query[A] www.buffer.com from 192.168.0.2\n"))
	f.Write([]byte("May 24 12:00:08 something dnsmasq[131]: query[A] api.buffer.com from 192.168.0.2\n"))
	f.Close()
	time.Sleep(time.Second)
	latest := store.GetLatestRequests(false)
	if len(*latest) != 3 {
		fmt.Printf("Failed: device count=%d\n", len(*latest))
		t.Fail()
	}
	for device, hosts := range *latest {
		fmt.Printf("Device: %v, hosts: %v\n", device, hosts)
		if len(*hosts) != 2 {
			t.Fail()
		}
	}
	os.Remove(path)
	os.Remove(db)
}
