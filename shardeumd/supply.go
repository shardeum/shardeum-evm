package shardeumd

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/shardeum/shardeum-evm/shardeumd/cmd/shardeumd/config"
)

const (
	// Base denom used in the chain
	BaseDenom = "ashm"
	// Cache TTL for supply calculations
	SupplyCacheTTL = 60 * time.Second
)

// CachedSupplyValue stores a supply value with its timestamp
type CachedSupplyValue struct {
	Value     *big.Int
	Timestamp time.Time
}

// Cache storage using atomic values for thread-safe access
var (
	totalSupplyCache       atomic.Value // stores *CachedSupplyValue
	totalSupplyMutex       sync.Mutex   // prevents cache stampede for total supply
	circulatingSupplyCache atomic.Value // stores *CachedSupplyValue
	circulatingSupplyMutex sync.Mutex   // prevents cache stampede for circulating supply
)

// getCachedValue retrieves a cached value if it's still valid
func getCachedValue(cache *atomic.Value) (*big.Int, bool) {
	cached := cache.Load()
	if cached == nil {
		return nil, false
	}

	cachedVal, ok := cached.(*CachedSupplyValue)
	if !ok || cachedVal == nil {
		return nil, false
	}

	// Check if cache is still valid
	if time.Since(cachedVal.Timestamp) > SupplyCacheTTL {
		return nil, false
	}

	// Return a copy to prevent mutation
	return new(big.Int).Set(cachedVal.Value), true
}

// setCachedValue stores a value in the cache with current timestamp
func setCachedValue(cache *atomic.Value, value *big.Int) {
	cache.Store(&CachedSupplyValue{
		Value:     new(big.Int).Set(value),
		Timestamp: time.Now(),
	})
}

// Module accounts to exclude from circulating supply
var excludedModules = map[string]bool{
//	"bonded_tokens_pool":     true,		//bonded will get counted in circulating supply
//	"not_bonded_tokens_pool": true,     //Holds tokens that are unbonding or delegated to unbonding/unbonded validators. counts for circulating supply
	"distribution":           true,		//rewards or commission waiting to be claimed
	"gov":                    true,
	"fee_collector":          true,
	"mint":                   true,
	"ibc-transfer":           true,
	"staking":                true,
	"transfer":               true,
	"erc20":                  true,
	"evm":                    true,
	"feemarket":              true,
	"precisebank":            true,
}

// Network-specific non-circulating addresses (foundation, team, ecosystem, sale cold storage)
// These are configured per network
var networkNonCircAddresses = map[string][]string{
	"shardeum_8118-1": { // mainnet
		"shardeum1lg3dd9d0zmvkpcrmrkr5guc6dn8s2k43gr3p6j", // Foundation Cold Storage
		"shardeum16r5ta5cpvcnkfgrkxvgnnnjwfd4eqlhend3ly8", // Team Cold Storage
		"shardeum1twv0y2s9426yqxmmh3u3r30q86ha8j0wcu7ajf", // Ecosystem Cold Storage
		"shardeum1uev04qcth9hgxkrvcs2dnntuyazlppg9nr8ul6", // Sale Cold Storage
	},
	"shardeum_8119-2": { // testnet
		"shardeum1lg3dd9d0zmvkpcrmrkr5guc6dn8s2k43gr3p6j",
		"shardeum16r5ta5cpvcnkfgrkxvgnnnjwfd4eqlhend3ly8",
		"shardeum1twv0y2s9426yqxmmh3u3r30q86ha8j0wcu7ajf",
		"shardeum1uev04qcth9hgxkrvcs2dnntuyazlppg9nr8ul6",
	},
	"shardeum_8119-3": { // devnet
		"shardeum1lg3dd9d0zmvkpcrmrkr5guc6dn8s2k43gr3p6j",
		"shardeum16r5ta5cpvcnkfgrkxvgnnnjwfd4eqlhend3ly8",
		"shardeum1twv0y2s9426yqxmmh3u3r30q86ha8j0wcu7ajf",
		"shardeum1uev04qcth9hgxkrvcs2dnntuyazlppg9nr8ul6",
	},
	"shardeum_8117-1": { // local
		// empty list
	},
}

// CalculateTotalSupply returns the total supply of SHM (whole number, not ashm)
func CalculateTotalSupply(sdkCtx sdk.Context, app *ShardeumApp) (result *big.Int, err error) {
	// Check cache first (fast path - no lock)
	if cached, ok := getCachedValue(&totalSupplyCache); ok {
		return cached, nil
	}

	// Cache miss - acquire lock
	totalSupplyMutex.Lock()
	defer totalSupplyMutex.Unlock()

	// Double-check cache after acquiring lock
	// Another goroutine might have computed it while we waited
	if cached, ok := getCachedValue(&totalSupplyCache); ok {
		return cached, nil
	}

	// Wrap in defer to catch any panics and convert to errors
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in CalculateTotalSupply: %v", r)
			result = big.NewInt(0)
		}
	}()

	// Validate inputs
	if app == nil {
		return big.NewInt(0), fmt.Errorf("app is nil")
	}

	// Validate context with panic protection
	var multiStoreErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				multiStoreErr = fmt.Errorf("panic accessing MultiStore: %v", r)
			}
		}()

		if sdkCtx.MultiStore() == nil {
			multiStoreErr = fmt.Errorf("SDK context multistore is nil")
		}
	}()

	if multiStoreErr != nil {
		return big.NewInt(0), multiStoreErr
	}

	// Get total supply for ashm denomination
	var totalSupply sdk.Coin
	var getSupplyErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				getSupplyErr = fmt.Errorf("panic in GetSupply: %v", r)
			}
		}()
		totalSupply = app.BankKeeper.GetSupply(sdkCtx, BaseDenom)
	}()

	if getSupplyErr != nil {
		return big.NewInt(0), getSupplyErr
	}

	var amountStr string
	var strErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				strErr = fmt.Errorf("panic accessing Amount.String(): %v", r)
			}
		}()

		if totalSupply.Amount.IsNil() {
			strErr = fmt.Errorf("amount is nil")
			return
		}

		amountStr = totalSupply.Amount.String()
	}()

	if strErr != nil {
		return big.NewInt(0), strErr
	}

	if amountStr == "" || amountStr == "<nil>" {
		return big.NewInt(0), fmt.Errorf("total supply amount string is empty or nil")
	}

	totalAshm := new(big.Int)
	totalAshm, ok := totalAshm.SetString(amountStr, 10)
	if !ok || totalAshm == nil {
		return big.NewInt(0), fmt.Errorf("failed to parse total supply amount: %s", amountStr)
	}

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(config.ShardeumChainDecimals())), nil)
	totalSHM := new(big.Int).Div(totalAshm, divisor)

	// Cache the result
	setCachedValue(&totalSupplyCache, totalSHM)

	return totalSHM, nil
}

// CalculateCirculatingSupply returns the circulating supply of SHM
func CalculateCirculatingSupply(sdkCtx sdk.Context, app *ShardeumApp) (result *big.Int, err error) {
	// Check cache first (fast path - no lock)
	if cached, ok := getCachedValue(&circulatingSupplyCache); ok {
		return cached, nil
	}

	// Cache miss - acquire lock to prevent stampede
	circulatingSupplyMutex.Lock()
	defer circulatingSupplyMutex.Unlock()

	// Double-check cache after acquiring lock
	// Another goroutine might have computed it while we waited
	if cached, ok := getCachedValue(&circulatingSupplyCache); ok {
		return cached, nil
	}

	// Wrap in defer to catch any panics and convert to errors
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in CalculateCirculatingSupply: %v", r)
			result = big.NewInt(0)
		}
	}()

	// Validate inputs
	if app == nil {
		return big.NewInt(0), fmt.Errorf("app is nil")
	}

	if sdkCtx.MultiStore() == nil {
		return big.NewInt(0), fmt.Errorf("SDK context multistore is nil")
	}

	var totalSupply sdk.Coin
	var getSupplyErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				getSupplyErr = fmt.Errorf("panic in GetSupply: %v", r)
			}
		}()
		totalSupply = app.BankKeeper.GetSupply(sdkCtx, BaseDenom)
	}()

	if getSupplyErr != nil {
		return big.NewInt(0), getSupplyErr
	}

	// Convert total supply to BigInt using safe string method with panic protection
	var totalSupplyStr string
	var strErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				strErr = fmt.Errorf("panic accessing Amount.String(): %v", r)
			}
		}()

		if totalSupply.Amount.IsNil() {
			strErr = fmt.Errorf("total supply amount is nil")
			return
		}

		totalSupplyStr = totalSupply.Amount.String()
	}()

	if strErr != nil {
		return big.NewInt(0), strErr
	}

	if totalSupplyStr == "" || totalSupplyStr == "<nil>" {
		return big.NewInt(0), fmt.Errorf("total supply string is empty or nil")
	}

	totalSupplyBigInt := new(big.Int)
	totalSupplyBigInt, ok := totalSupplyBigInt.SetString(totalSupplyStr, 10)
	if !ok || totalSupplyBigInt == nil {
		return big.NewInt(0), fmt.Errorf("failed to parse total supply: %s", totalSupplyStr)
	}

	// 2. Get balances of excluded module accounts
	excludedModuleBalance := big.NewInt(0)
	for moduleName := range excludedModules {
		moduleAddr := app.AccountKeeper.GetModuleAddress(moduleName)
		if moduleAddr == nil || len(moduleAddr) == 0 {
			// Module doesn't exist in this chain, skip it
			continue
		}

		// Wrap balance query in panic recovery
		var balance sdk.Coin
		var queryErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					queryErr = fmt.Errorf("panic querying module %s: %v", moduleName, r)
				}
			}()
			balance = app.BankKeeper.GetBalance(sdkCtx, moduleAddr, BaseDenom)
		}()

		if queryErr != nil {
			return big.NewInt(0), fmt.Errorf("failed to query balance for module %s: %w", moduleName, queryErr)
		}

		if balance.Amount.IsNil() || balance.Amount.IsZero() {
			// Zero balance is valid, skip
			continue
		}

		balanceStr := balance.Amount.String()
		if balanceStr == "" || balanceStr == "<nil>" {
			return big.NewInt(0), fmt.Errorf("module %s balance string is empty or nil", moduleName)
		}

		balanceBigInt := new(big.Int)
		balanceBigInt, ok := balanceBigInt.SetString(balanceStr, 10)
		if !ok || balanceBigInt == nil {
			return big.NewInt(0), fmt.Errorf("failed to parse module %s balance: %s", moduleName, balanceStr)
		}
		excludedModuleBalance = new(big.Int).Add(excludedModuleBalance, balanceBigInt)
	}

	// 3. Get balances of non-circulating addresses (foundation, team, ecosystem, sale cold storage)
	chainID := sdkCtx.ChainID()
	nonCircAddresses := networkNonCircAddresses[chainID]

	// Debug: Log chain ID and address count
	_ = fmt.Sprintf("ChainID: %s, Cold storage addresses: %d", chainID, len(nonCircAddresses))

	excludedAddressBalance := big.NewInt(0)
	for _, addrStr := range nonCircAddresses {
		// Process each cold storage address with error handling
		addr, err := sdk.AccAddressFromBech32(addrStr)
		if err != nil {
			// Skip invalid addresses
			continue
		}

		// Wrap balance query in panic recovery
		var balance sdk.Coin
		var queryErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					queryErr = fmt.Errorf("panic querying balance for %s: %v", addrStr, r)
				}
			}()
			balance = app.BankKeeper.GetBalance(sdkCtx, addr, BaseDenom)
		}()

		if queryErr != nil {
			return big.NewInt(0), fmt.Errorf("failed to query balance for address %s: %w", addrStr, queryErr)
		}

		if balance.Amount.IsNil() || balance.Amount.IsZero() {
			continue
		}

		balanceStr := balance.Amount.String()
		if balanceStr == "" || balanceStr == "<nil>" {
			return big.NewInt(0), fmt.Errorf("balance string for address %s is empty or nil", addrStr)
		}

		balanceBigInt := new(big.Int)
		balanceBigInt, ok := balanceBigInt.SetString(balanceStr, 10)
		if !ok || balanceBigInt == nil {
			return big.NewInt(0), fmt.Errorf("failed to parse balance for address %s: %s", addrStr, balanceStr)
		}
		excludedAddressBalance = new(big.Int).Add(excludedAddressBalance, balanceBigInt)
	}

	// 4. Get community pool balance (unclaimed rewards in distribution module)
	communityPoolBalance := big.NewInt(0)
	var poolErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				poolErr = fmt.Errorf("panic querying community pool: %v", r)
			}
		}()

		// Query community pool from distribution module
		pool, err := app.DistrKeeper.FeePool.Get(sdkCtx)
		if err != nil {
			poolErr = fmt.Errorf("failed to get fee pool: %w", err)
			return
		}

		if pool.CommunityPool != nil && len(pool.CommunityPool) > 0 {
			for _, coin := range pool.CommunityPool {
				if coin.Denom == BaseDenom {
					poolStr := coin.Amount.TruncateInt().String()
					if poolStr == "" || poolStr == "<nil>" {
						poolErr = fmt.Errorf("community pool balance string is empty or nil")
						return
					}
					poolBigInt := new(big.Int)
					poolBigInt, ok := poolBigInt.SetString(poolStr, 10)
					if !ok || poolBigInt == nil {
						poolErr = fmt.Errorf("failed to parse community pool balance: %s", poolStr)
						return
					}
					communityPoolBalance = poolBigInt
					break
				}
			}
		}
	}()

	if poolErr != nil {
		return big.NewInt(0), poolErr
	}

	// 5. Calculate circulating supply
	// Circulating = Total - ExcludedModules - CommunityPool - ExcludedAddresses
	circulating := new(big.Int).Sub(totalSupplyBigInt, excludedModuleBalance)
	circulating = new(big.Int).Sub(circulating, communityPoolBalance)
	circulating = new(big.Int).Sub(circulating, excludedAddressBalance)

	// Ensure non-negative
	if circulating.Sign() < 0 {
		circulating = big.NewInt(0)
	}

	// Convert from ashm to SHM (divide by 10^decimals)
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(config.ShardeumChainDecimals())), nil)
	circulatingSHM := new(big.Int).Div(circulating, divisor)

	// Cache the result
	setCachedValue(&circulatingSupplyCache, circulatingSHM)

	return circulatingSHM, nil
}

// SupplyResponse represents the JSON response format for supply endpoints
type SupplyResponse struct {
	Result string `json:"result"`
}

// formatSupplyResponse formats the supply response based on the Accept header
// Supports both "application/json" and "text/plain" (default)
func formatSupplyResponse(w http.ResponseWriter, r *http.Request, supply *big.Int, supplyType string) {
	// Check Accept header or format query parameter
	acceptHeader := r.Header.Get("Accept")
	formatParam := r.URL.Query().Get("format")

	// Determine response format
	useJSON := false
	if formatParam == "json" || strings.Contains(acceptHeader, "application/json") {
		useJSON = true
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if useJSON {
		// Return JSON response with 18 decimal places
		decimals := config.ShardeumChainDecimals()
		decimalStr := fmt.Sprintf("%s.%0*d", supply.String(), decimals, 0)

		response := SupplyResponse{
			Result: decimalStr,
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode JSON: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		// Return plain text response (default) - whole number without decimals
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", supply.String())
	}
}
