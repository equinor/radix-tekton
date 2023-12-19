package internal

import (
	"errors"

	"github.com/equinor/radix-operator/pkg/apis/config/dnsalias"
)

// ValidateReservedDNSAliases Validate reserved DNS aliases
func ValidateReservedDNSAliases(dnsConfig *dnsalias.DNSConfig) error {
	var errs []error
	if len(dnsConfig.ReservedAppDNSAliases) == 0 {
		errs = append(errs, ErrMissingReservedDNSAliasesForRadixApplications)
	}
	if len(dnsConfig.ReservedDNSAliases) == 0 {
		errs = append(errs, ErrMissingReservedDNSAliasesForRadixServices)
	}
	return errors.Join(errs...)
}
