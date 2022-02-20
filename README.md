# caddy-geofence

A caddy module for IP geofencing your caddy web server using freegeoip.app

![Build Status](https://github.com/circa10a/caddy-geofence/workflows/deploy/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/circa10a/caddy-geofence)](https://goreportcard.com/report/github.com/circa10a/caddy-geofence)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/circa10a/caddy-geofence?style=plastic)
![Docker Pulls](https://img.shields.io/docker/pulls/circa10a/caddy-geofence?style=plastic)

![alt text](https://user-images.githubusercontent.com/1128849/36338535-05fb646a-136f-11e8-987b-e6901e717d5a.png)

- [caddy-geofence](#caddy-geofence)
  - [Usage](#usage)
    - [Build with caddy](#build-with-caddy)
    - [Docker](#docker)
  - [Caddyfile example](#caddyfile-example)
  - [Development](#development)
    - [Run](#run)
    - [Build](#build)

## Usage

1. For an IP that is not within the geofence, `403` will be returned on the matching route.
2. An API token from [freegeoip.app](https://freegeoip.app/) is **required** to run this module.

> Free tier includes 15,000 requests per hour

### Build with caddy

```shell
# build module with caddy
xcaddy build --with github.com/circa10a/caddy-geofence
```

### Docker

```shell
docker run --net host -v /your/Caddyfile:/etc/caddy/Caddyfile -e FREEGEOIP_API_TOKEN -p 80:80 -p 443:443 circa10a/caddy-geofence
```

## Caddyfile example

```
{
	debug
	order geofence before respond
}

:80

route /* {
	geofence {
		# cache_ttl is the duration to store ip addresses and if they are within proximity or not to increase performance
		# Cache for 7 days, valid time units are "ms", "s", "m", "h"
		# Not specifying a TTL sets no expiration on cached items and will live until restart
		cache_ttl 168h

		# freegeoip.app API token, this example reads from an environment variable
		freegeoip_api_token {$FREEGEOIP_API_TOKEN}

		// radius is the distance of the geofence in kilometers
		// If not supplied, will default to 0.0 kilometers
		// 1.0 => 1.0 kilometers
		radius 1.0

		# allow_private_ip_addresses is a boolean for whether or not to allow private ip ranges
		# such as 192.X, 172.X, 10.X, [::1] (localhost)
		# false by default
		# Some cellular networks doing NATing with 172.X addresses, in which case, you may not want to allow
		allow_private_ip_addresses true

		# allowlist is a list of IP addresses that will not be checked for proximity and will be allowed to access the server
		allowlist 206.189.205.251 206.189.205.252

		# status_code is the HTTP response code that is returned if IP address is not within proximity. Default is 403
		status_code 403
	}
}

log {
	output stdout
}
```

## Development

Requires [xcaddy](https://caddyserver.com/docs/build#xcaddy) to be installed

### Run

```shell
export FREEGEOIP_API_TOKEN=<token>
make run
```

### Build

```shell
make build
```
