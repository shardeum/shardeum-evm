package evm

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestCheckDualLayerFee(t *testing.T) {
	tests := []struct {
		name              string
		fee               sdkmath.LegacyDec
		nodeMinGasPrice   sdkmath.LegacyDec
		globalMinGasPrice sdkmath.LegacyDec
		gasLimit          sdkmath.LegacyDec
		isLondon          bool
		expectError       bool
		expectedErrorMsg  string
	}{
		{
			name:              "fee meets both requirements",
			fee:               sdkmath.LegacyNewDec(1000),
			nodeMinGasPrice:   sdkmath.LegacyNewDec(10),
			globalMinGasPrice: sdkmath.LegacyNewDec(5),
			gasLimit:          sdkmath.LegacyNewDec(100),
			isLondon:          true,
			expectError:       false,
		},
		{
			name:              "fee below node minimum",
			fee:               sdkmath.LegacyNewDec(500),
			nodeMinGasPrice:   sdkmath.LegacyNewDec(10),
			globalMinGasPrice: sdkmath.LegacyNewDec(5),
			gasLimit:          sdkmath.LegacyNewDec(100),
			isLondon:          true,
			expectError:       true,
			expectedErrorMsg:  "fee below node minimum",
		},
		{
			name:              "fee below global minimum",
			fee:               sdkmath.LegacyNewDec(800),
			nodeMinGasPrice:   sdkmath.LegacyNewDec(5),
			globalMinGasPrice: sdkmath.LegacyNewDec(10),
			gasLimit:          sdkmath.LegacyNewDec(100),
			isLondon:          true,
			expectError:       true,
			expectedErrorMsg:  "fee below global minimum",
		},
		{
			name:              "zero node minimum allows transaction",
			fee:               sdkmath.LegacyNewDec(500),
			nodeMinGasPrice:   sdkmath.LegacyZeroDec(),
			globalMinGasPrice: sdkmath.LegacyNewDec(5),
			gasLimit:          sdkmath.LegacyNewDec(100),
			isLondon:          true,
			expectError:       false,
		},
		{
			name:              "zero global minimum allows transaction",
			fee:               sdkmath.LegacyNewDec(500),
			nodeMinGasPrice:   sdkmath.LegacyNewDec(5),
			globalMinGasPrice: sdkmath.LegacyZeroDec(),
			gasLimit:          sdkmath.LegacyNewDec(100),
			isLondon:          true,
			expectError:       false,
		},
		{
			name:              "both minimums zero allows any fee",
			fee:               sdkmath.LegacyNewDec(1),
			nodeMinGasPrice:   sdkmath.LegacyZeroDec(),
			globalMinGasPrice: sdkmath.LegacyZeroDec(),
			gasLimit:          sdkmath.LegacyNewDec(100),
			isLondon:          true,
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckDualLayerFee(tt.fee, tt.nodeMinGasPrice, tt.globalMinGasPrice, tt.gasLimit, tt.isLondon)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
