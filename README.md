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

> Free tier includes 100 requests per month

### Build with caddy

```shell
# build module with caddy
xcaddy build --with github.com/circa10a/caddy-geofence
```

### Docker

```shell
docker run -v /your/Caddyfile:/etc/caddy/Caddyfile -e FREEGEOIP_API_TOKEN -p 80:80 -p 443:443 circa10a/caddy-geofence
```

## Caddyfile example

```
{
        debug
        order geofence before respond
}

localhost:80

route /* {
        geofence {
                # Cache ip addresses and if they are within proximity or not
                cache_ttl 5m
                # freegeoip.app API token, this example reads from an environment variable
                freegeoip_api_token {$FREEGEOIP_API_TOKEN}
                # Proximity
                # 0 - 111 km
                # 1 - 11.1 km
                # 2 - 1.11 km
                # 3 111 meters
                # 4 11.1 meters
                # 5 1.11 meters
                sensitivity 3
        }
}
```

## Development

Requires [xcaddy](https://caddyserver.com/docs/build#xcaddy) to be installed

### Run

```shell
export IP_STACK_API_TOKEN=<token>
make run
```

### Build

```shell
make build
```
