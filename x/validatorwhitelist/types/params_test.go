package types

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParamsTestSuite struct {
	suite.Suite
}

func TestParamsTestSuite(t *testing.T) {
	suite.Run(t, new(ParamsTestSuite))
}

func (suite *ParamsTestSuite) TestParamsValidate() {
	testCases := []struct {
		name    string
		params  Params
		expPass bool
	}{
		{
			"valid params - disabled",
			Params{
				Enabled:   false,
				Allowlist: []string{},
			},
			true,
		},
		{
			"valid params - enabled with validators",
			Params{
				Enabled:   true,
				Allowlist: []string{"shardeumvaloper1abc123", "shardeumvaloper1def456"},
			},
			true,
		},
		{
			"valid params - enabled with empty allowlist",
			Params{
				Enabled:   true,
				Allowlist: []string{},
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.params.Validate()
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *ParamsTestSuite) TestDefaultParams() {
	params := DefaultParams()
	suite.Require().False(params.Enabled)
	suite.Require().Empty(params.Allowlist)
}

func (suite *ParamsTestSuite) TestIsValidatorAllowed() {
	testCases := []struct {
		name           string
		params         Params
		validatorAddr  string
		expectedResult bool
	}{
		{
			"disabled whitelist - validator in allowlist",
			Params{
				Enabled:   false,
				Allowlist: []string{"shardeumvaloper1abc123"},
			},
			"shardeumvaloper1abc123",
			true, // Should return true when disabled (all validators allowed)
		},
		{
			"disabled whitelist - validator not in allowlist",
			Params{
				Enabled:   false,
				Allowlist: []string{"shardeumvaloper1abc123"},
			},
			"shardeumvaloper1xyz789",
			true, // Should return true when disabled (all validators allowed)
		},
		{
			"enabled whitelist - validator in allowlist",
			Params{
				Enabled:   true,
				Allowlist: []string{"shardeumvaloper1abc123"},
			},
			"shardeumvaloper1abc123",
			true,
		},
		{
			"enabled whitelist - validator not in allowlist",
			Params{
				Enabled:   true,
				Allowlist: []string{"shardeumvaloper1abc123"},
			},
			"shardeumvaloper1xyz789",
			false,
		},
		{
			"enabled whitelist - empty allowlist",
			Params{
				Enabled:   true,
				Allowlist: []string{},
			},
			"shardeumvaloper1abc123",
			false,
		},
		{
			"case insensitive test",
			Params{
				Enabled:   true,
				Allowlist: []string{"shardeumvaloper1abc123"},
			},
			"ShardeumValoper1abc123",
			true, // Should be case insensitive
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := tc.params.IsValidatorAllowed(tc.validatorAddr)
			suite.Require().Equal(tc.expectedResult, result)
		})
	}
}
