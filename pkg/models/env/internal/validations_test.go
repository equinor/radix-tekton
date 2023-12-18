package internal_test

import (
	"errors"
	"testing"

	"github.com/equinor/radix-operator/pkg/apis/config/dnsalias"
	"github.com/equinor/radix-tekton/pkg/models/env/internal"
	"github.com/stretchr/testify/assert"
)

func TestValidateReservedDNSAliases(t *testing.T) {
	tests := []struct {
		name       string
		dnsConfig  *dnsalias.DNSConfig
		wantErrors []error
	}{
		{
			name: "nil",
			dnsConfig: &dnsalias.DNSConfig{
				ReservedAppDNSAliases: nil,
				ReservedDNSAliases:    nil,
			},
			wantErrors: []error{internal.ErrMissingReservedDNSAliasesForRadixApplications, internal.ErrMissingReservedDNSAliasesForRadixServices},
		},
		{
			name: "empty",
			dnsConfig: &dnsalias.DNSConfig{
				ReservedAppDNSAliases: make(map[string]string),
				ReservedDNSAliases:    make([]string, 0),
			},
			wantErrors: []error{internal.ErrMissingReservedDNSAliasesForRadixApplications, internal.ErrMissingReservedDNSAliasesForRadixServices},
		},
		{
			name: "radix apps missing",
			dnsConfig: &dnsalias.DNSConfig{
				ReservedAppDNSAliases: make(map[string]string),
				ReservedDNSAliases:    []string{"service1"},
			},
			wantErrors: []error{internal.ErrMissingReservedDNSAliasesForRadixApplications},
		},
		{
			name: "radix services missing",
			dnsConfig: &dnsalias.DNSConfig{
				ReservedAppDNSAliases: map[string]string{"app1": "some-app"},
				ReservedDNSAliases:    make([]string, 0),
			},
			wantErrors: []error{internal.ErrMissingReservedDNSAliasesForRadixServices},
		},
		{
			name: "no errors",
			dnsConfig: &dnsalias.DNSConfig{
				ReservedAppDNSAliases: map[string]string{"app1": "some-app"},
				ReservedDNSAliases:    []string{"service1"},
			},
			wantErrors: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := internal.ValidateReservedDNSAliases(tt.dnsConfig)
			assert.Equal(t, errors.Join(tt.wantErrors...), err)
		})
	}
}
