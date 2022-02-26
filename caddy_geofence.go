package caddygeofence

import (
	"fmt"
	"net"
	"net/http"
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
	// 403
	defaultStatusCode = http.StatusForbidden
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
	// remote_ip is the IP address to geofence against
	// Not specifying this field results in geofencing the public address of the machine caddy is running on
	RemoteIP string `json:"remote_ip,omitempty"`
	// allowlist is a list of IP addresses that will not be checked for proximity and will be allowed to access the server
	Allowlist []string `json:"allowlist,omitempty"`
	// status_code is the HTTP response code that is returned if IP address is not within proximity. Default is 403
	StatusCode int `json:"status_code,omitempty"`
	// cache_ttl is string parameter for caching ip addresses with their allowed/not allowed state
	// Not specifying a TTL sets no expiration on cached items and will live until restart
	// Valid time units are "ms", "s", "m", "h"
	CacheTTL time.Duration `json:"cache_ttl,omitempty"`
	// radius is the distance of the geofence in kilometers
	// If not supplied, will default to 0.0 kilometers
	// 1.0 => 1.0 kilometers
	Radius float64 `json:"radius"`
	// allow_private_ip_addresses is a boolean for whether or not to allow private ip ranges
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

// Provision implements caddy.Provisioner.
func (cg *CaddyGeofence) Provision(ctx caddy.Context) error {
	// Instantiate logger
	cg.logger = caddy.Log()

	// Verify API Token is set
	if cg.FreeGeoIPAPIToken == "" {
		return fmt.Errorf("freegeoip_api_token: freegeoip API token not set")
	}

	// Set cache to never expire if not set
	if cg.CacheTTL == 0 {
		cg.CacheTTL = defaultCacheTTL
	}

	// Set default status code if not set (403)
	if cg.StatusCode == 0 {
		cg.StatusCode = defaultStatusCode
	}

	// Setup client
	geofenceClient, err := geofence.New(&geofence.Config{
		IPAddress:               cg.RemoteIP,
		Token:                   cg.FreeGeoIPAPIToken,
		Radius:                  cg.Radius,
		AllowPrivateIPAddresses: cg.AllowPrivateIPAddresses,
		CacheTTL:                cg.CacheTTL,
	})
	if err != nil {
		return err
	}

	cg.GeofenceClient = geofenceClient
	return nil
}

// Validate validates that the module has a usable config.
func (cg CaddyGeofence) Validate() error {
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (cg CaddyGeofence) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Get host address, can  contain a port so we make sure we strip that off
	remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return err
	}

	// Check if ip address is in allowlist
	inAllowlist := strInSlice(remoteAddr, cg.Allowlist)

	// Debug private address/allowlist rules
	cg.logger.Debug(loggerNamespace,
		zap.String("remote_addr", remoteAddr),
		zap.Bool("is_private_address_allowed", cg.AllowPrivateIPAddresses),
		zap.Bool("is_in_allowlist", inAllowlist),
	)

	// If ip address is in allowlist, continue
	if inAllowlist {
		return next.ServeHTTP(w, r)
	}

	// Check if ip address is nearby
	isAddressNear, err := cg.GeofenceClient.IsIPAddressNear(remoteAddr)
	if err != nil {
		return err
	}

	// Debug geofencing
	cg.logger.Debug(loggerNamespace,
		zap.String("remote_addr", remoteAddr),
		zap.Bool("is_ip_address_near", isAddressNear),
	)

	// If remote address is not nearby, reject the request
	if !isAddressNear {
		return caddyhttp.Error(cg.StatusCode, nil)
	}

	return next.ServeHTTP(w, r)
}

// strInSlice returns true if string is in slice
func strInSlice(str string, list []string) bool {
	for _, item := range list {
		if str == item {
			return true
		}
	}
	return false
}

// Interface guards
var (
	_ caddy.Provisioner           = (*CaddyGeofence)(nil)
	_ caddy.Validator             = (*CaddyGeofence)(nil)
	_ caddyhttp.MiddlewareHandler = (*CaddyGeofence)(nil)
	_ caddyfile.Unmarshaler       = (*CaddyGeofence)(nil)
)
