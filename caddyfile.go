package caddygeofence

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (cg *CaddyGeofence) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if d.NextArg() {
			return d.ArgErr()
		}
		// Validate args
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			switch d.Val() {
			case "cache_ttl":
				if !d.NextArg() {
					return d.ArgErr()
				}
				// Setup cache
				cacheTTL, err := time.ParseDuration(d.Val())
				if err != nil {
					return err
				}
				cg.CacheTTL = cacheTTL
			case "freegeoip_api_token":
				if !d.NextArg() {
					return d.ArgErr()
				}
				cg.FreeGeoIPAPIToken = d.Val()
			case "remote_IP":
				if !d.NextArg() {
					return d.ArgErr()
				}
				if net.ParseIP(d.Val()) == nil {
					return fmt.Errorf("remote_ip: invalid IPv4 address provided")
				}
				cg.RemoteIP = d.Val()
			case "sensitivity":
				if !d.NextArg() {
					return d.ArgErr()
				}
				sensitivity, err := strconv.Atoi(d.Val())
				if err != nil {
					return err
				}
				cg.Sensitivity = sensitivity
			}
		}
	}
	return nil
}

// parseCaddyfile unmarshals tokens from h into a new Middleware.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var cg CaddyGeofence
	err := cg.UnmarshalCaddyfile(h.Dispenser)
	return cg, err
}
