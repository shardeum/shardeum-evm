package common

import (
	"encoding/json"
	"fmt"
	"strconv"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// SignedEIP712Tx is an interface for extracting messages from signed EIP-712 transactions
type SignedEIP712Tx interface {
	GetMsgs() []json.RawMessage
}

// BuildMessages parses raw messages from a signed EIP-712 transaction and converts
// them to SDK messages.
//
// Supported message types today:
// - cosmos-sdk/MsgDelegate
// - cosmos-sdk/MsgUndelegate
// - cosmos-sdk/MsgVote (governance voting)
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

		case "cosmos-sdk/MsgVote":
			msg, err := parseMsgVote(msgWrapper.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse MsgVote: %w", err)
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

// parseMsgVote parses a MsgVote (governance vote) message from JSON.
// This expects the legacy gov MsgVote shape used by the EIP-712 encoding:
//
//	{
//	  "proposal_id": "5",
//	  "voter": "shardeum1...",
//	  "option": 1
//	}
func parseMsgVote(value json.RawMessage) (*govtypes.MsgVote, error) {
	var voteValue struct {
		ProposalID string `json:"proposal_id"`
		Voter      string `json:"voter"`
		Option     int32  `json:"option"`
	}

	if err := json.Unmarshal(value, &voteValue); err != nil {
		return nil, err
	}

	if voteValue.Voter == "" {
		return nil, fmt.Errorf("voter address is required")
	}

	voterAddr, err := sdk.AccAddressFromBech32(voteValue.Voter)
	if err != nil {
		return nil, fmt.Errorf("invalid voter bech32 address %q: %w", voteValue.Voter, err)
	}

	proposalID, err := strconv.ParseUint(voteValue.ProposalID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid proposal_id %q: %w", voteValue.ProposalID, err)
	}

	// Map the numeric option directly onto the legacy gov VoteOption enum:
	// 1 = Yes, 2 = Abstain, 3 = No, 4 = NoWithVeto.
	option := govtypes.VoteOption(voteValue.Option)

	return govtypes.NewMsgVote(
		voterAddr,
		proposalID,
		option,
	), nil
}
