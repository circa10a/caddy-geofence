package caddygeofence

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/circa10a/go-geofence"
)

const (
	// Infinite
	defaultCacheTTL = -1
	// 100m
	defaultSensitivity = 3
)

// CaddyGeofence implements IP geofencing functionality.
type CaddyGeofence struct {
	// CacheTTL is string parameter for caching ip addresses with their allowed/not allowed state
	// Not specifying a TTL sets no expiration on cached items and will live until restart
	// Time format examples include 10s, 10m, 10h, 10d, 10h45m, 1w
	CacheTTL time.Duration `json:"cache_ttl,omitempty"`
	// IPStackAPIToken is REQUIRED and is the API token from ipstack.com
	// Free tier includes 100 requests per month
	IPStackAPIToken string `json:"ipstack_api_token,omitempty"`
	// RemoteIP is the IP address to geofence against
	// Not specifying this field results in geofencing the public address of the machine caddy is running on
	RemoteIP string `json:"remote_ip,omitempty"`
	// Sensitivity is a 0-5 scale of the geofence proximity
	// 0 - 111 km
	// 1 - 11.1 km
	// 2 - 1.11 km
	// 3 111 meters
	// 4 11.1 meters
	// 5 1.11 meters
	Sensitivity    int `json:"sensitivity,omitempty"`
	GeofenceClient *geofence.Geofence
}

func init() {
	caddy.RegisterModule(CaddyGeofence{})
	httpcaddyfile.RegisterHandlerDirective("geofence", parseCaddyfile)
}

// CaddyModule returns the Caddy module information.
func (CaddyGeofence) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.geofence",
		New: func() caddy.Module { return new(CaddyGeofence) },
	}
}

// isPrivateAddress checks if remote address is from known private ip space
func isPrivateAddress(addr string) bool {
	return strings.HasPrefix(addr, "192.") ||
		strings.HasPrefix(addr, "172.") ||
		strings.HasPrefix(addr, "10.") ||
		strings.HasPrefix(addr, "[::1]")
}

// Provision implements caddy.Provisioner.
func (cg *CaddyGeofence) Provision(ctx caddy.Context) error {
	if cg.IPStackAPIToken == "" {
		return fmt.Errorf("ipstack_api_token: ipstack API token not set")
	}
	// Set cache to never expire if not specified
	if cg.CacheTTL == 0 {
		cg.CacheTTL = defaultCacheTTL
	}
	// Set sensitivity to mid range if not set
	if cg.Sensitivity == 0 {
		cg.Sensitivity = defaultSensitivity
	}

	// Setup client
	geofenceClient, err := geofence.New(cg.RemoteIP, cg.IPStackAPIToken, cg.Sensitivity)
	if err != nil {
		return err
	}
	cg.GeofenceClient = geofenceClient
	// Setup cache
	cg.GeofenceClient.CreateCache(cg.CacheTTL)
	return nil
}

// Validate validates that the module has a usable config.
func (cg CaddyGeofence) Validate() error {
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (cg CaddyGeofence) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	remoteAddr := r.RemoteAddr
	// If known private ip, skip
	if isPrivateAddress(remoteAddr) {
		return next.ServeHTTP(w, r)
	}
	// Check if address is nearby
	isAddressNear, err := cg.GeofenceClient.IsIPAddressNear(remoteAddr)
	if err != nil {
		return err
	}
	// If remote address is not nearby, reject the request
	if !isAddressNear {
		return caddyhttp.Error(http.StatusForbidden, nil)
	}
	return next.ServeHTTP(w, r)
}

// Interface guards
var (
	_ caddy.Provisioner           = (*CaddyGeofence)(nil)
	_ caddy.Validator             = (*CaddyGeofence)(nil)
	_ caddyhttp.MiddlewareHandler = (*CaddyGeofence)(nil)
	_ caddyfile.Unmarshaler       = (*CaddyGeofence)(nil)
)
