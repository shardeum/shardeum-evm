package types

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type GenesisTestSuite struct {
	suite.Suite
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) TestDefaultGenesisState() {
	genState := DefaultGenesisState()
	suite.Require().NotNil(genState)
	suite.Require().False(genState.Params.Enabled)
	suite.Require().Empty(genState.Params.Allowlist)
}

func (suite *GenesisTestSuite) TestNewGenesisState() {
	params := Params{
		Enabled:   true,
		Allowlist: []string{"shardeumvaloper1abc123", "shardeumvaloper1def456"},
	}

	genState := NewGenesisState(params)
	suite.Require().NotNil(genState)
	suite.Require().True(genState.Params.Enabled)
	suite.Require().Equal([]string{"shardeumvaloper1abc123", "shardeumvaloper1def456"}, genState.Params.Allowlist)
}

func (suite *GenesisTestSuite) TestGenesisStateValidate() {
	testCases := []struct {
		name    string
		genState GenesisState
		expPass bool
	}{
		{
			"valid genesis state - disabled",
			GenesisState{
				Params: Params{
					Enabled:   false,
					Allowlist: []string{},
				},
			},
			true,
		},
		{
			"valid genesis state - enabled with validators",
			GenesisState{
				Params: Params{
					Enabled:   true,
					Allowlist: []string{"shardeumvaloper1abc123", "shardeumvaloper1def456"},
				},
			},
			true,
		},
		{
			"valid genesis state - enabled with empty allowlist",
			GenesisState{
				Params: Params{
					Enabled:   true,
					Allowlist: []string{},
				},
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.genState.Validate()
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
