# Network Log Monitor

Monitors the DNS entries in a log file and sends an email update periodically
showing the hosts that have been requested by each device

Devices or Hosts can be ignored using the Web UI

## Building

[go-bindata](https://github.com/jteeuwen/go-bindata) is used to package the templates

``` go-bindata -pkg notify -o notify/templates.go templates/email-content.template ```

``` go-bindata -pkg ui -o ui/templates.go templates/ignored-devices.template templates/authorized-hosts.template ```

``` go build ```

## Running

``` network-log-monitor [-cfg <path/to/config.json>] [<path/to/log>] ```

## Configuring

```
{
  "DbURL":"the/path/to/the/database/file",
  "HTTPHost":"host:port (used to start the server)",
  "HTTPAddress":"http://host:port (used for links)",
  "LogPath":"the/path/to/the/log/file",
  "MailInterval":1440 (in minutes),
  "MailConfig":{
    "From":"from@address.com",
    "To":"to@address.com",
    "Subject":"Email Subject",
    "SMTPServer":"smtp.server.com",
    "SMTPPort":25,
    "SMTPUser":"username@address.com",
    "SMTPPassword":"password123"
  }
}
```
