package caddygeofence

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/circa10a/go-geofence"
	"go.uber.org/zap"
)

const (
	// Infinite
	defaultCacheTTL = -1
	// 100m
	defaultSensitivity = 3
	// Logger namespace string
	loggerNamespace = "geofence"
)

// CaddyGeofence implements IP geofencing functionality. https://github.com/circa10a/caddy-geofence
type CaddyGeofence struct {
	logger         *zap.Logger
	GeofenceClient *geofence.Geofence
	// freegeoip_api_token is REQUIRED and is an API token from freegeoip.app
	// Free tier includes 15000 requests per hour
	FreeGeoIPAPIToken string `json:"freegeoip_api_token,omitempty"`
	// RemoteIP is the IP address to geofence against
	// Not specifying this field results in geofencing the public address of the machine caddy is running on
	RemoteIP string `json:"remote_ip,omitempty"`
	// CacheTTL is string parameter for caching ip addresses with their allowed/not allowed state
	// Not specifying a TTL sets no expiration on cached items and will live until restart
	// Valid time units are "ms", "s", "m", "h"
	CacheTTL time.Duration `json:"cache_ttl,omitempty"`
	// Sensitivity is a 0-5 scale of the geofence proximity
	// 0 - 111 km
	// 1 - 11.1 km
	// 2 - 1.11 km
	// 3 111 meters
	// 4 11.1 meters
	// 5 1.11 meters
	Sensitivity int `json:"sensitivity,omitempty"`
	// AllowPrivateIPAddresses is a boolean for whether or not to allow private ip ranges
	// such as 192.X, 172.X, 10.X, [::1] (localhost)
	// false by default
	// Some cellular networks doing NATing with 172.X addresses, in which case, you may not want to allow
	AllowPrivateIPAddresses bool `json:"allow_private_ip_addresses"`
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
	// Instantiate logger
	cg.logger = caddy.Log()

	// Verify API Token is set
	if cg.FreeGeoIPAPIToken == "" {
		return fmt.Errorf("freegeoip_api_token: freegeoip API token not set")
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
	geofenceClient, err := geofence.New(cg.RemoteIP, cg.FreeGeoIPAPIToken, cg.Sensitivity)
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
	// Debug private addresses
	cg.logger.Debug(loggerNamespace,
		zap.String("remote_addr", remoteAddr),
		zap.Bool("is_private_address", isPrivateAddress(remoteAddr)),
		zap.Bool("is_private_address_allowed", cg.AllowPrivateIPAddresses),
	)

	// If known private ip and config says to allow
	if isPrivateAddress(remoteAddr) && cg.AllowPrivateIPAddresses {
		return next.ServeHTTP(w, r)
	}

	// Check if address is nearby
	isAddressNear, err := cg.GeofenceClient.IsIPAddressNear(remoteAddr)
	if err != nil {
		// go-geofence will complain about [::1] not being a a valid ip which is correct, but we want to ignore it
		// to prevent more errors in logs
		if !errors.Is(err, &geofence.ErrInvalidIPAddress{}) && !strings.HasPrefix(remoteAddr, "[::1]") {
			return err
		}
	}
	// Debug geofencing
	// Debug private addresses
	cg.logger.Debug(loggerNamespace,
		zap.String("remote_addr", remoteAddr),
		zap.Bool("is_ip_address_near", isAddressNear),
	)

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
