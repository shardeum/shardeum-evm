package shardeumd

import (
	"google.golang.org/grpc"

	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
)

// noMsgServerVestingModule wraps the standard x/auth/vesting AppModule but skips
// registering its Msg service. With no handler wired, the vesting messages
// (MsgCreateVestingAccount / MsgCreatePeriodicVestingAccount /
// MsgCreatePermanentLockedAccount) fail at routing across EVERY dispatch path —
// direct txs, authz MsgExec, and group/gov msg execution — so there is no path to
// create a vesting account at runtime.
//
// Everything else is inherited unchanged from the embedded module: interface /
// account-type registration (via AppModuleBasic), no-op genesis, and name/ordering.
// Vesting account TYPES therefore remain decodable — this only removes runtime
// creation, not genesis-defined accounts.
//
// Why: Shardeum's deployed (pre-v0.6.0) x/vm balance mirror reconciles against
// SpendableCoin and does not carry a LockedCoins snapshot. Vesting accounts are the
// only source of LockedCoins, so blocking their creation keeps LockedCoins provably
// zero and the mirror a no-op — avoiding a risky backport of SetBalanceWithLocked.
// Remove this shim if/when the v0.6.0 locked-balance mirror is adopted and vesting is
// intentionally supported.
type noMsgServerVestingModule struct {
	vesting.AppModule
}

// RegisterServices intentionally does nothing, so no vesting Msg handler is wired.
func (noMsgServerVestingModule) RegisterServices(grpc.ServiceRegistrar) error {
	return nil
}
