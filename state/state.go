package state

import (
	"encoding/json"
	"log"
	"sort"
	"time"

	"github.com/boltdb/bolt"
)

const devicesBucket = "devices"
const ignoredBucket = "ignored"
const authorizedBucket = "authorized"

// Host : A host a device has requested
type Host struct {
	Host  string
	Times *[]*time.Time
}

// AddRequest : Add a request for this host
func (host *Host) AddRequest(at *time.Time) {
	*host.Times = append(*host.Times, at)
}

// Device : A device that has connected
type Device struct {
	At       *time.Time
	Hostname string
	Mac      string
	IP       string
	Requests *map[string]*Host
}

// Name : the name of the device
func (device *Device) Name() string {
	return device.Hostname
}

// AddRequest : associates a request with this device
func (device *Device) AddRequest(at *time.Time, host string) {
	log.Printf("Adding request: %s to %s\n", host, device.IP)
	if existing, exists := (*device.Requests)[host]; exists {
		existing.AddRequest(at)
	} else {
		list := append(make([]*time.Time, 0), at)
		(*device.Requests)[host] = &Host{host, &list}
	}
}

func (device *Device) marshall() []byte {
	data, _ := json.Marshal(device)
	return data
}

type byTime []*Device

func (devices byTime) Len() int           { return len(devices) }
func (devices byTime) Swap(i, j int)      { devices[i], devices[j] = devices[j], devices[i] }
func (devices byTime) Less(i, j int) bool { return devices[i].At.After(*devices[j].At) }

// Store : all the state that is managed
type Store struct {
	db           *bolt.DB
	ignored      map[string]bool
	authorized   map[string]bool
	devicesByIP  map[string]*Device
	devicesByMAC map[string]*Device
}

// IgnoreDevice : adds the device to the list of ignored devices
func (store *Store) IgnoreDevice(mac string) {
	store.ignored[mac] = true
	err := persistKey(store.db, ignoredBucket, mac)
	logError("Error ignoring device: %v\n", err)
}

// UnIgnoreDevice : adds the device to the list of ignored devices
func (store *Store) UnIgnoreDevice(ip string) {
	delete(store.ignored, ip)
	err := removeKey(store.db, ignoredBucket, ip)
	logError("Error unignoring device: %v\n", err)
}

// GetIgnoredDevices : gets the list of ignored devices
func (store *Store) GetIgnoredDevices() *map[string]bool {
	return &store.ignored
}

// AuthoriseHost : adds the host to the list of authorized hosts
func (store *Store) AuthoriseHost(host string) {
	store.authorized[host] = true
	err := persistKey(store.db, authorizedBucket, host)
	logError("Error authorising device: %v\n", err)
}

// DeauthoriseHost : adds the host to the list of authorized hosts
func (store *Store) DeauthoriseHost(host string) {
	delete(store.authorized, host)
	err := removeKey(store.db, authorizedBucket, host)
	logError("Error deauthorising device: %v\n", err)
}

// GetAuthorisedHosts :
func (store *Store) GetAuthorisedHosts() *map[string]bool {
	return &store.authorized
}

// AddDevice : adds the device to the list of devices
func (store *Store) AddDevice(at *time.Time, hostname string, ip string, mac string) *Device {
	hosts := make(map[string]*Host, 0)
	device := &Device{at, hostname, mac, ip, &hosts}
	store.devicesByIP[ip] = device
	store.devicesByMAC[mac] = device
	err := persistDevice(store.db, device)
	logError("Error adding device: %v\n", err)
	return device
}

// FindDeviceByIP : Find the last device to use this IP
func (store *Store) FindDeviceByIP(ip string) *Device {
	return store.devicesByIP[ip]
}

// FindDeviceByMac : Find the device by MAC Address
func (store *Store) FindDeviceByMac(mac string) *Device {
	return store.devicesByMAC[mac]
}

// GetLatestRequests : Get a map of all the devices
func (store *Store) GetLatestRequests(reset bool) *map[*Device]*map[string]*Host {
	requests := make(map[*Device]*map[string]*Host, 0)
	for _, device := range store.devicesByMAC {
		requests[device] = device.Requests
		if reset {
			hosts := make(map[string]*Host, 0)
			device.Requests = &hosts
		}
	}
	return &requests
}

// NewStore : Create a new empty Store
func NewStore(url string) (*Store, error) {
	db, err := bolt.Open(url, 0600, nil)
	if err != nil {
		return nil, err
	}
	ignored := make(map[string]bool, 0)
	authorized := make(map[string]bool, 0)
	devices := make([]*Device, 0)
	db.Update(func(tx *bolt.Tx) error {
		loadMap(tx, ignoredBucket, ignored)
		loadMap(tx, authorizedBucket, authorized)
		loadDevices(tx, &devices)
		return nil
	})
	byIP := make(map[string]*Device, 0)
	byMAC := make(map[string]*Device, 0)
	sort.Sort(byTime(devices))
	for _, device := range devices {
		log.Printf("Loading device: %v\n", device)
		byIP[device.IP] = device
		byMAC[device.Mac] = device
	}
	log.Printf("Loaded ignored: %d authorized: %d devices: %d\n", len(ignored), len(authorized), len(byIP))
	return &Store{db, ignored, authorized, byIP, byMAC}, nil
}

func loadMap(tx *bolt.Tx, bucket string, result map[string]bool) {
	if bucket, err := tx.CreateBucketIfNotExists([]byte(bucket)); err == nil {
		bucket.ForEach(func(k []byte, v []byte) error {
			result[string(k)] = (v[0] == 1)
			return nil
		})
	}
}

func loadDevices(tx *bolt.Tx, devices *[]*Device) {
	if bucket, err := tx.CreateBucketIfNotExists([]byte(devicesBucket)); err == nil {
		bucket.ForEach(func(k []byte, v []byte) error {
			hosts := make(map[string]*Host, 0)
			device := Device{&time.Time{}, "Unknown", "00:00:00:00:00:00", "0.0.0.0", &hosts}
			json.Unmarshal(v, &device)
			*devices = append(*devices, &device)
			return nil
		})
	}
}

func persistDevice(db *bolt.DB, device *Device) error {
	log.Printf("Persisting device: %v\n", *device)
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(devicesBucket))
		return bucket.Put([]byte(device.Mac), device.marshall())
	})
}

func removeKey(db *bolt.DB, bucket string, key string) error {
	log.Printf("Removing key: %v\n", key)
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket))
		return bucket.Delete([]byte(key))
	})
}

func persistKey(db *bolt.DB, bucket string, key string) error {
	log.Printf("Persisting key: %v\n", key)
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket))
		return bucket.Put([]byte(key), []byte{1})
	})
}

func logError(msg string, err error) {
	if err != nil {
		log.Printf(msg, err)
	}
}
