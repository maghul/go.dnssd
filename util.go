package dnssd

import (
	"fmt"
)

// Concatenate a three-part domain name (as provided to the response funcs) into a properly-escaped full domain name.
func ConstructFullName(serviceName, regType, domain string) string {
	return fmt.Sprintf("%s.%s.%s.", serviceName, regType, domain)
}
