package common

import (
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// SignedEIP712Tx is an interface for extracting messages from signed EIP-712 transactions
type SignedEIP712Tx interface {
	GetMsgs() []json.RawMessage
}

// BuildMessages parses raw messages from a signed EIP-712 transaction and converts them to SDK messages.
// Supports multiple message types: MsgDelegate and MsgUndelegate.
func BuildMessages(signedTx SignedEIP712Tx) ([]sdk.Msg, error) {
	var msgs []sdk.Msg

	for _, rawMsg := range signedTx.GetMsgs() {
		var msgWrapper struct {
			Type  string          `json:"type"`
			Value json.RawMessage `json:"value"`
		}

		if err := json.Unmarshal(rawMsg, &msgWrapper); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message wrapper: %w", err)
		}

		switch msgWrapper.Type {
		case "cosmos-sdk/MsgDelegate":
			msg, err := parseMsgDelegate(msgWrapper.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse MsgDelegate: %w", err)
			}
			msgs = append(msgs, msg)

		case "cosmos-sdk/MsgUndelegate":
			msg, err := parseMsgUndelegate(msgWrapper.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse MsgUndelegate: %w", err)
			}
			msgs = append(msgs, msg)

		default:
			return nil, fmt.Errorf("unsupported message type: %s", msgWrapper.Type)
		}
	}

	return msgs, nil
}

// parseMsgDelegate parses a MsgDelegate message from JSON
func parseMsgDelegate(value json.RawMessage) (*stakingtypes.MsgDelegate, error) {
	var delegateValue struct {
		DelegatorAddress string `json:"delegator_address"`
		ValidatorAddress string `json:"validator_address"`
		Amount           struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"amount"`
	}

	if err := json.Unmarshal(value, &delegateValue); err != nil {
		return nil, err
	}

	amount, ok := sdkmath.NewIntFromString(delegateValue.Amount.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", delegateValue.Amount.Amount)
	}

	return stakingtypes.NewMsgDelegate(
		delegateValue.DelegatorAddress,
		delegateValue.ValidatorAddress,
		sdk.NewCoin(delegateValue.Amount.Denom, amount),
	), nil
}

// parseMsgUndelegate parses a MsgUndelegate message from JSON
func parseMsgUndelegate(value json.RawMessage) (*stakingtypes.MsgUndelegate, error) {
	var undelegateValue struct {
		DelegatorAddress string `json:"delegator_address"`
		ValidatorAddress string `json:"validator_address"`
		Amount           struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"amount"`
	}

	if err := json.Unmarshal(value, &undelegateValue); err != nil {
		return nil, err
	}

	amount, ok := sdkmath.NewIntFromString(undelegateValue.Amount.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", undelegateValue.Amount.Amount)
	}

	return stakingtypes.NewMsgUndelegate(
		undelegateValue.DelegatorAddress,
		undelegateValue.ValidatorAddress,
		sdk.NewCoin(undelegateValue.Amount.Denom, amount),
	), nil
}

