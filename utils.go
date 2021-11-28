package caddygeofence

import "strings"

// isPrivateAddress checks if remote address is from known private ip space
func isPrivateAddress(addr string) bool {
	return strings.HasPrefix(addr, "192.") ||
		strings.HasPrefix(addr, "172.") ||
		strings.HasPrefix(addr, "10.") ||
		strings.HasPrefix(addr, "::1")
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
