package types

import (
	"fmt"
	"strings"
)

// DefaultParams returns default validator whitelist parameters
func DefaultParams() *Params {
	return &Params{
		Enabled:   false,
		Allowlist: []string{},
	}
}

// Validate validates the parameters
func (p *Params) Validate() error {
	for i, addr := range p.Allowlist {
		if addr == "" {
			return fmt.Errorf("validator address at index %d cannot be empty", i)
		}
		// Additional validation can be added here if needed
	}
	return nil
}

// IsValidatorAllowed checks if a validator is in the allowlist
func (p *Params) IsValidatorAllowed(validatorAddr string) bool {
	if !p.Enabled {
		return true // If whitelist is disabled, all validators are allowed
	}
	
	for _, allowedAddr := range p.Allowlist {
		if strings.EqualFold(allowedAddr, validatorAddr) {
			return true
		}
	}
	return false
}
