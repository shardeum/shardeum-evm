package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/shardeum/shardeum-evm/x/validatorwhitelist/types"
)

var _ types.MsgServer = &Keeper{}

// UpdateWhitelist updates the validator whitelist
func (k Keeper) UpdateWhitelist(goCtx context.Context, msg *types.MsgUpdateWhitelist) (*types.MsgUpdateWhitelistResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Create new params
	params := &types.Params{
		Enabled:   msg.Enabled,
		Allowlist: msg.Allowlist,
	}

	// Validate the new parameters
	if err := params.Validate(); err != nil {
		return nil, errorsmod.Wrapf(err, "invalid parameters")
	}

	// Set the new parameters
	k.SetParams(ctx, params)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUpdateWhitelist,
			sdk.NewAttribute(types.AttributeKeyEnabled, fmt.Sprintf("%t", params.Enabled)),
			sdk.NewAttribute(types.AttributeKeyAllowlistCount, fmt.Sprintf("%d", len(params.Allowlist))),
		),
	)

	return &types.MsgUpdateWhitelistResponse{}, nil
}

// AddValidator adds a validator to the whitelist
func (k Keeper) AddValidator(goCtx context.Context, msg *types.MsgAddValidator) (*types.MsgAddValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get current parameters
	params := k.GetParams(ctx)

	// Check if validator is already in the allowlist
	for _, addr := range params.Allowlist {
		if addr == msg.Validator {
			return nil, errorsmod.Wrapf(errors.ErrInvalidRequest, "validator %s is already in the allowlist", msg.Validator)
		}
	}

	// Add validator to allowlist
	params.Allowlist = append(params.Allowlist, msg.Validator)

	// Validate the updated parameters
	if err := params.Validate(); err != nil {
		return nil, errorsmod.Wrapf(err, "invalid parameters")
	}

	// Set the updated parameters
	k.SetParams(ctx, params)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAddValidator,
			sdk.NewAttribute(types.AttributeKeyValidator, msg.Validator),
		),
	)

	return &types.MsgAddValidatorResponse{}, nil
}

// RemoveValidator removes a validator from the whitelist
func (k Keeper) RemoveValidator(goCtx context.Context, msg *types.MsgRemoveValidator) (*types.MsgRemoveValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get current parameters
	params := k.GetParams(ctx)

	// Find and remove validator from allowlist
	found := false
	newAllowlist := make([]string, 0, len(params.Allowlist))
	for _, addr := range params.Allowlist {
		if addr != msg.Validator {
			newAllowlist = append(newAllowlist, addr)
		} else {
			found = true
		}
	}

	if !found {
		return nil, errorsmod.Wrapf(errors.ErrInvalidRequest, "validator %s is not in the allowlist", msg.Validator)
	}

	// Update allowlist
	params.Allowlist = newAllowlist

	// Validate the updated parameters
	if err := params.Validate(); err != nil {
		return nil, errorsmod.Wrapf(err, "invalid parameters")
	}

	// Set the updated parameters
	k.SetParams(ctx, params)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRemoveValidator,
			sdk.NewAttribute(types.AttributeKeyValidator, msg.Validator),
		),
	)

	return &types.MsgRemoveValidatorResponse{}, nil
}