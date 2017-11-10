package state

import (
	"os"
	"testing"
	"time"
)

func TestAuthoriseHost(t *testing.T) {
	store, _ := NewStore("/tmp/authorise-host")
	store.AuthoriseHost("added")
	store.AuthoriseHost("added.and.removed")
	store.DeauthoriseHost("added.and.removed")
	store.DeauthoriseHost("removed")
	store.AuthoriseHost("added.and.removed.and.added.again")
	store.DeauthoriseHost("added.and.removed.and.added.again")
	store.AuthoriseHost("added.and.removed.and.added.again")
	store.Close()

	newStore, _ := NewStore("/tmp/authorise-host")
	defer newStore.Close()
	defer os.Remove("/tmp/authorise-host")
	hosts := *newStore.GetAuthorisedHosts()
	_, added := hosts["added"]
	_, addedAndRemoved := hosts["added.and.removed"]
	_, removed := hosts["removed"]
	_, addedAndRemovedAndAdded := hosts["added.and.removed.and.added.again"]
	if !added || addedAndRemoved || removed || !addedAndRemovedAndAdded {
		t.Fail()
	}
}

func TestIgnoreDevice(t *testing.T) {
	store, _ := NewStore("/tmp/ignore-device")
	store.IgnoreDevice("added")
	store.IgnoreDevice("added.and.removed")
	store.UnIgnoreDevice("added.and.removed")
	store.UnIgnoreDevice("removed")
	store.IgnoreDevice("added.and.removed.and.added.again")
	store.UnIgnoreDevice("added.and.removed.and.added.again")
	store.IgnoreDevice("added.and.removed.and.added.again")
	store.Close()

	newStore, _ := NewStore("/tmp/ignore-device")
	defer newStore.Close()
	defer os.Remove("/tmp/ignore-device")
	hosts := *newStore.GetIgnoredDevices()
	_, added := hosts["added"]
	_, addedAndRemoved := hosts["added.and.removed"]
	_, removed := hosts["removed"]
	_, addedAndRemovedAndAdded := hosts["added.and.removed.and.added.again"]
	if !added || addedAndRemoved || removed || !addedAndRemovedAndAdded {
		t.Fail()
	}
}

func TestAddDeviceAndRequest(t *testing.T) {
	store, _ := NewStore("/tmp/add-device")
	defer store.Close()
	defer os.Remove("/tmp/add-device")
	at := time.Now()
	device := store.AddDevice(&at, "hostname", "127.0.0.1", "AA:BB:CC:DD:EE:FF")
	at = at.Add(time.Second)
	device.AddRequest(&at, "www.google.com")
	at = at.Add(time.Second)
	device.AddRequest(&at, "www.another.com")
	at = at.Add(time.Second)
	device = store.AddDevice(&at, "another", "127.0.0.2", "AA:BB:CC:DD:EE:GG")
	at = at.Add(time.Second)
	device.AddRequest(&at, "www.first.com")
	at = at.Add(time.Second)
	device.AddRequest(&at, "www.first.com")
	store.GetLatestRequests(false)
	if countRequests(store.GetLatestRequests(true)) != 3 ||
		countRequests(store.GetLatestRequests(false)) != 0 ||
		store.FindDeviceByIP("127.0.0.2").Name() != "another" {
		t.Fail()
	}
}

func TestInvalidUrl(t *testing.T) {
	_, err := NewStore("/tmp/a path/that does/not exist")
	if err == nil {
		t.Fail()
	}
}

func TestClosedStore(t *testing.T) {
	store, _ := NewStore("/tmp/closed-store")
	store.Close()
	store.AuthoriseHost("a.host")
	os.Remove("/tmp/closed-store")
}

func countRequests(devices *map[*Device]*map[string]*Host) int {
	count := 0
	for _, hosts := range *devices {
		count += len(*hosts)
	}
	return count
}
