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
			case "ipbase_api_token":
				if !d.NextArg() {
					return d.ArgErr()
				}
				cg.IPBaseAPIToken = d.Val()
			case "remote_ip":
				if !d.NextArg() {
					return d.ArgErr()
				}
				if net.ParseIP(d.Val()) == nil {
					return fmt.Errorf("remote_ip: invalid IP address provided")
				}
				cg.RemoteIP = d.Val()
			case "allowlist":
				cg.Allowlist = d.RemainingArgs()
				if len(cg.Allowlist) == 0 {
					return d.ArgErr()
				}
			case "status_code":
				if !d.NextArg() {
					return d.ArgErr()
				}
				statusCode, err := strconv.Atoi(d.Val())
				if err != nil {
					return err
				}
				cg.StatusCode = statusCode
			case "radius":
				if !d.NextArg() {
					return d.ArgErr()
				}
				radius, err := strconv.ParseFloat(d.Val(), 64)
				if err != nil {
					return err
				}
				cg.Radius = radius
			case "allow_private_ip_addresses":
				if !d.NextArg() {
					return d.ArgErr()
				}
				allowPrivateIPAddresses, err := strconv.ParseBool(d.Val())
				if err != nil {
					return err
				}
				cg.AllowPrivateIPAddresses = allowPrivateIPAddresses
			case "redis_enabled":
				if !d.NextArg() {
					return d.ArgErr()
				}
				redisEnabled, err := strconv.ParseBool(d.Val())
				if err != nil {
					return err
				}
				cg.RedisEnabled = redisEnabled
			case "redis_username":
				if !d.NextArg() {
					return d.ArgErr()
				}
				cg.RedisUsername = d.Val()
			case "redis_password":
				if !d.NextArg() {
					return d.ArgErr()
				}
				cg.RedisPassword = d.Val()
			case "redis_addr":
				if !d.NextArg() {
					return d.ArgErr()
				}
				cg.RedisAddr = d.Val()
			case "redis_db":
				if !d.NextArg() {
					return d.ArgErr()
				}
				redisDB, err := strconv.Atoi(d.Val())
				if err != nil {
					return err
				}
				cg.RedisDB = redisDB
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
