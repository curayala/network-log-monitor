package ui

import (
	"html/template"
	"net/http"

	"github.com/tmullender/network-log-monitor/state"
)

var authorizedHostsFile, _ = Asset("templates/authorized-hosts.template")
var authorizedHosts = template.Must(template.New("authorized-hosts").Parse(string(authorizedHostsFile)))
var ignoredDevicesFile, _ = Asset("templates/ignored-devices.template")
var ignoredDevices = template.Must(template.New("ignored-devices").Parse(string(ignoredDevicesFile)))

// Root : Returns a handler for the root URL
func Root() func(resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("Welcome to Network Log Monitor"))
	}
}

// GetAuthorizedHosts : Returns a handler for rendering the authorized hosts page
func GetAuthorizedHosts(store *state.Store) func(resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		authorizedHosts.Execute(resp, store.GetAuthorisedHosts())
	}
}

// AddAuthorizedHosts : Returns a handler for adding an authorized host
func AddAuthorizedHosts(store *state.Store) func(resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		host := req.FormValue("host")
		store.AuthoriseHost(host)
		authorizedHosts.Execute(resp, store.GetAuthorisedHosts())
	}
}

// RemoveAuthorizedHosts : Returns a handler for removing an authorized host
func RemoveAuthorizedHosts(store *state.Store) func(resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		host := req.FormValue("host")
		store.DeauthoriseHost(host)
		authorizedHosts.Execute(resp, store.GetAuthorisedHosts())
	}
}

// GetIgnoredDevices : Returns a handler for rendering ignored devices
func GetIgnoredDevices(store *state.Store) func(resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		ignoredDevices.Execute(resp, store.GetIgnoredDevices())
	}
}

// AddIgnoredDevice : Returns a handler for adding an ignored device
func AddIgnoredDevice(store *state.Store) func(resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		mac := req.FormValue("mac")
		store.IgnoreDevice(mac)
		ignoredDevices.Execute(resp, store.GetIgnoredDevices())
	}
}

// RemoveIgnoredDevice : Returns a handler for removing an ignored device
func RemoveIgnoredDevice(store *state.Store) func(resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		mac := req.FormValue("mac")
		store.UnIgnoreDevice(mac)
		ignoredDevices.Execute(resp, store.GetIgnoredDevices())
	}
}
