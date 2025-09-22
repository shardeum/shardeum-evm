package interfaces

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

// MockValidatorWhitelistKeeper is a mock implementation for testing
type MockValidatorWhitelistKeeper struct {
	enabled           bool
	allowedValidators map[string]bool
}

func NewMockValidatorWhitelistKeeper(enabled bool, allowedValidators []string) ValidatorWhitelistKeeper {
	allowed := make(map[string]bool)
	for _, addr := range allowedValidators {
		allowed[addr] = true
	}
	return &MockValidatorWhitelistKeeper{
		enabled:           enabled,
		allowedValidators: allowed,
	}
}

func (m *MockValidatorWhitelistKeeper) IsWhitelistEnabled(ctx sdk.Context) bool {
	return m.enabled
}

func (m *MockValidatorWhitelistKeeper) CanValidatorReceiveDelegation(ctx sdk.Context, validatorAddr string) bool {
	if !m.enabled {
		return true
	}
	return m.allowedValidators[validatorAddr]
}

func TestConfigValidatorWhitelistKeeper(t *testing.T) {
	ctx := sdk.Context{} // In real tests, you'd create a proper context

	t.Run("disabled whitelist allows all delegations", func(t *testing.T) {
		keeper := NewAppConfigValidatorWhitelistKeeper(false, []string{})
		
		assert.False(t, keeper.IsWhitelistEnabled(ctx))
		assert.True(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1abc123"))
		assert.True(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1def456"))
	})

	t.Run("enabled whitelist only allows delegations to listed validators", func(t *testing.T) {
		allowedValidators := []string{
			"cosmosvaloper1abc123",
			"cosmosvaloper1def456",
		}
		keeper := NewAppConfigValidatorWhitelistKeeper(true, allowedValidators)
		
		assert.True(t, keeper.IsWhitelistEnabled(ctx))
		assert.True(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1abc123"))
		assert.True(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1def456"))
		assert.False(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1ghi789"))
	})

	t.Run("handles empty allowlist", func(t *testing.T) {
		keeper := NewAppConfigValidatorWhitelistKeeper(true, []string{})
		
		assert.True(t, keeper.IsWhitelistEnabled(ctx))
		assert.False(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1abc123"))
	})

	t.Run("trims whitespace from validator addresses", func(t *testing.T) {
		allowedValidators := []string{
			"  cosmosvaloper1abc123  ",
			"cosmosvaloper1def456",
		}
		keeper := NewAppConfigValidatorWhitelistKeeper(true, allowedValidators)
		
		assert.True(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1abc123"))
		assert.True(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1def456"))
	})
}

func TestMockValidatorWhitelistKeeper(t *testing.T) {
	ctx := sdk.Context{} // In real tests, you'd create a proper context

	t.Run("mock keeper works correctly", func(t *testing.T) {
		allowedValidators := []string{
			"cosmosvaloper1abc123",
			"cosmosvaloper1def456",
		}
		keeper := NewMockValidatorWhitelistKeeper(true, allowedValidators)
		
		assert.True(t, keeper.IsWhitelistEnabled(ctx))
		assert.True(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1abc123"))
		assert.True(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1def456"))
		assert.False(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1ghi789"))
	})
}

// Example of how you could test the ante decorator with different keeper implementations
func TestValidatorWhitelistDecoratorWithDifferentKeepers(t *testing.T) {
	// This demonstrates how the interface makes it easy to test with different implementations
	// You could test with:
	// 1. ConfigValidatorWhitelistKeeper (config-based)
	// 2. MockValidatorWhitelistKeeper (for unit tests)
	// 3. OnChainValidatorWhitelistKeeper (on-chain state)
	// 4. Any other implementation you create
	
	ctx := sdk.Context{}
	
	// Test with config-based keeper
	configKeeper := NewAppConfigValidatorWhitelistKeeper(true, []string{"cosmosvaloper1abc123"})
	assert.True(t, configKeeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1abc123"))
	
	// Test with mock keeper
	mockKeeper := NewMockValidatorWhitelistKeeper(true, []string{"cosmosvaloper1def456"})
	assert.True(t, mockKeeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1def456"))
	
	// Both implement the same interface, so they can be used interchangeably
	keepers := []ValidatorWhitelistKeeper{configKeeper, mockKeeper}
	
	for i, keeper := range keepers {
		t.Logf("Testing keeper %d", i)
		assert.True(t, keeper.IsWhitelistEnabled(ctx))
		// Each keeper has its own logic, but they all implement the same interface
	}
}

// Test delegation-specific functionality
func TestValidatorDelegationControl(t *testing.T) {
	ctx := sdk.Context{}

	t.Run("allows delegation to whitelisted validators", func(t *testing.T) {
		keeper := NewAppConfigValidatorWhitelistKeeper(true, []string{"cosmosvaloper1abc123"})
		
		assert.True(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1abc123"))
	})

	t.Run("blocks delegation to non-whitelisted validators", func(t *testing.T) {
		keeper := NewAppConfigValidatorWhitelistKeeper(true, []string{"cosmosvaloper1abc123"})
		
		assert.False(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1xyz789"))
	})

	t.Run("allows all delegations when whitelist is disabled", func(t *testing.T) {
		keeper := NewAppConfigValidatorWhitelistKeeper(false, []string{})
		
		assert.True(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1abc123"))
		assert.True(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1xyz789"))
	})

	t.Run("handles case sensitivity correctly", func(t *testing.T) {
		keeper := NewAppConfigValidatorWhitelistKeeper(true, []string{"cosmosvaloper1abc123"})
		
		assert.True(t, keeper.CanValidatorReceiveDelegation(ctx, "cosmosvaloper1abc123"))
		assert.False(t, keeper.CanValidatorReceiveDelegation(ctx, "CosmosValoper1abc123"))
		assert.False(t, keeper.CanValidatorReceiveDelegation(ctx, "COSMOSVALOPER1ABC123"))
	})
}
