package internal

import "errors"

var (
	ErrMissingReservedDNSAliasesForRadixApplications = errors.New("missing DNS aliases, reserved for Radix platform Radix application")
	ErrMissingReservedDNSAliasesForRadixServices     = errors.New("missing DNS aliases, reserved for Radix platform services")
)
