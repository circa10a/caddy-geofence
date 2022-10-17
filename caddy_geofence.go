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
	"github.com/go-redis/redis/v9"
	"go.uber.org/zap"
)

const (
	// Infinite
	defaultCacheTTL = -1
	// 403
	defaultStatusCode = http.StatusForbidden
	// Localhost for default redis instance
	defaultRedisAddr = "localhost:6379"
	// Logger namespace string
	loggerNamespace = "geofence"
)

// CaddyGeofence implements IP geofencing functionality. https://github.com/circa10a/caddy-geofence
type CaddyGeofence struct {
	logger         *zap.Logger
	geofenceClient *geofence.Geofence
	// ipbase_api_token is REQUIRED and is an API token ipbase.com.
	// Free tier includes 150 requests per month.
	IPBaseAPIToken string `json:"ipbase_api_token,omitempty"`
	// remote_ip is the IP address to geofence against.
	// Not specifying this field results in geofencing the public address of the machine caddy is running on.
	RemoteIP string `json:"remote_ip,omitempty"`
	// allowlist is a list of IP addresses that will not be checked for proximity and will be allowed to access the server.
	Allowlist []string `json:"allowlist,omitempty"`
	// status_code is the HTTP response code that is returned if IP address is not within proximity. Default is 403.
	StatusCode int `json:"status_code,omitempty"`
	// cache_ttl is string parameter for caching ip addresses with their allowed/not allowed state.
	// Not specifying a TTL sets no expiration on cached items and will live until restart.
	// Valid time units are "ms", "s", "m", "h".
	// In-memory cache is used if redis is not enabled.
	CacheTTL time.Duration `json:"cache_ttl,omitempty"`
	// radius is the distance of the geofence in kilometers.
	// If not supplied, will default to 0.0 kilometers.
	// 1.0 => 1.0 kilometers.
	Radius float64 `json:"radius"`
	// allow_private_ip_addresses is a boolean for whether or not to allow private ip ranges
	// such as 192.X, 172.X, 10.X, [::1] (localhost). Default is false.
	// Some cellular networks doing NATing with 172.X addresses, in which case, you may not want to allow.
	AllowPrivateIPAddresses bool `json:"allow_private_ip_addresses"`
	// redis_enabled uses redis for caching. Default is false.
	RedisEnabled bool `json:"redis_enabled,omitempty"`
	// redis_username is the username to connect to a redis instance. Default is "".
	RedisUsername string `json:"redis_username,omitempty"`
	// redis_password is the password to connect to a redis instance. Default is "".
	RedisPassword string `json:"redis_password,omitempty"`
	// redis_addr is the address to connect to a redis instance. Default is localhost:6379.
	RedisAddr string `json:"redis_addr,omitempty"`
	// redis_db is the db id. Default is 0.
	RedisDB int `json:"redis_db,omitempty"`
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
	if cg.IPBaseAPIToken == "" {
		return fmt.Errorf("ipbase_api_token: ipbase.com API token not set")
	}

	// Set cache to never expire if not set
	if cg.CacheTTL == 0 {
		cg.CacheTTL = defaultCacheTTL
	}

	// Set default status code if not set (403)
	if cg.StatusCode == 0 {
		cg.StatusCode = defaultStatusCode
	}

	// Setup base client options
	geofenceConfig := &geofence.Config{
		IPAddress:               cg.RemoteIP,
		Token:                   cg.IPBaseAPIToken,
		Radius:                  cg.Radius,
		AllowPrivateIPAddresses: cg.AllowPrivateIPAddresses,
		CacheTTL:                cg.CacheTTL,
	}

	// Setup redis
	// Set default redis addr if empty
	if cg.RedisAddr == "" {
		cg.RedisAddr = defaultRedisAddr
	}

	// If redis is enabled, set the options for go-geofence to create the client
	if cg.RedisEnabled {
		geofenceConfig.RedisOptions = &redis.Options{
			Addr:     cg.RedisAddr,
			Username: cg.RedisUsername,
			Password: cg.RedisPassword,
			DB:       cg.RedisDB,
		}
	}

	geofenceClient, err := geofence.New(geofenceConfig)
	if err != nil {
		return err
	}

	cg.geofenceClient = geofenceClient
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
	isAddressNear, err := cg.geofenceClient.IsIPAddressNear(remoteAddr)
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
