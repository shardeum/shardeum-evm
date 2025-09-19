package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	amino = codec.NewLegacyAmino()
	// ModuleCdc references the global validator whitelist module codec. Note, the codec should
	// ONLY be used in certain instances of tests and for JSON encoding.
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// AminoCdc is an amino codec created to support amino JSON compatible msgs.
	//
	// For now, we'll leave this. PENDING to check if we can remove it.
	AminoCdc = codec.NewAminoCodec(amino) //nolint:staticcheck
)

const (
	// Amino names
	updateWhitelistName = "shardeum/validatorwhitelist/MsgUpdateWhitelist"
	addValidatorName    = "shardeum/validatorwhitelist/MsgAddValidator"
	removeValidatorName = "shardeum/validatorwhitelist/MsgRemoveValidator"
)

// NOTE: This is required for the GetSignBytes function
func init() {
	RegisterLegacyAminoCodec(amino)
	amino.Seal()
}

// RegisterInterfaces registers the client interfaces to protobuf Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgUpdateWhitelist{},
		&MsgAddValidator{},
		&MsgRemoveValidator{},
	)
	// Note: Service descriptor registration is handled by the generated protobuf files
}

// RegisterLegacyAminoCodec required for EIP-712
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateWhitelist{}, updateWhitelistName, nil)
	cdc.RegisterConcrete(&MsgAddValidator{}, addValidatorName, nil)
	cdc.RegisterConcrete(&MsgRemoveValidator{}, removeValidatorName, nil)
}