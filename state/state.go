package state

import (
	"encoding/json"
	"log"
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
	log.Printf("Adding request: %s\n", host)
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
	persistKey(store.db, ignoredBucket, mac)
}

// UnIgnoreDevice : adds the device to the list of ignored devices
func (store *Store) UnIgnoreDevice(ip string) {
	delete(store.ignored, ip)
	removeKey(store.db, ignoredBucket, ip)
}

// GetIgnoredDevices : gets the list of ignored devices
func (store *Store) GetIgnoredDevices() *map[string]bool {
	return &store.ignored
}

// AuthoriseHost : adds the host to the list of authorized hosts
func (store *Store) AuthoriseHost(host string) {
	store.authorized[host] = true
	persistKey(store.db, authorizedBucket, host)
}

// DeauthoriseHost : adds the host to the list of authorized hosts
func (store *Store) DeauthoriseHost(host string) {
	delete(store.authorized, host)
	removeKey(store.db, authorizedBucket, host)
}

// GetAuthorisedHosts :
func (store *Store) GetAuthorisedHosts() *map[string]bool {
	return &store.authorized
}

// AddDevice : adds the device to the list of devices
func (store *Store) AddDevice(hostname string, ip string, mac string) *Device {
	hosts := make(map[string]*Host, 0)
	device := &Device{hostname, ip, mac, &hosts}
	store.devicesByIP[ip] = device
	store.devicesByMAC[mac] = device
	persistDevice(store.db, device)
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
func (store *Store) GetLatestRequests() *map[*Device]*map[string]*Host {
	requests := make(map[*Device]*map[string]*Host, 0)
	for _, device := range store.devicesByMAC {
		requests[device] = device.Requests
		hosts := make(map[string]*Host, 0)
		device.Requests = &hosts
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
	byMAC := make(map[string]*Device, 0)
	db.Update(func(tx *bolt.Tx) error {
		loadMap(tx, ignoredBucket, ignored)
		loadMap(tx, authorizedBucket, authorized)
		loadDevices(tx, byMAC)
		return nil
	})
	byIP := make(map[string]*Device, 0)
	for _, device := range byMAC {
		byIP[device.IP] = device
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

func loadDevices(tx *bolt.Tx, devices map[string]*Device) {
	if bucket, err := tx.CreateBucketIfNotExists([]byte(devicesBucket)); err == nil {
		bucket.ForEach(func(k []byte, v []byte) error {
			device := Device{}
			json.Unmarshal(v, device)
			devices[string(k)] = &device
			return nil
		})
	}
}

func persistDevice(db *bolt.DB, device *Device) {
	log.Printf("Persisting device: %v\n", *device)
	db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(devicesBucket))
		bucket.Put([]byte(device.Mac), device.marshall())
		return nil
	})
}

func removeKey(db *bolt.DB, bucket string, key string) {
	log.Printf("Removing key: %v\n", key)
	db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket))
		bucket.Delete([]byte(key))
		return nil
	})
}

func persistKey(db *bolt.DB, bucket string, key string) {
	log.Printf("Persisting key: %v\n", key)
	db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket))
		bucket.Put([]byte(key), []byte{1})
		return nil
	})
}