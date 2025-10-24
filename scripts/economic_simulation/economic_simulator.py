#!/usr/bin/env python3
"""
Economic Parameter Simulator for Shardeum Cosmos Network

This script simulates the Cosmos SDK mint module inflation algorithm
to evaluate the impact of different economic parameters on rewards, APY,
and token minting.

Based on Cosmos SDK mint module logic:
https://github.com/cosmos/cosmos-sdk/tree/main/x/mint
"""

import math
from dataclasses import dataclass
from typing import Dict, Any


# Constants
ASHM_PER_SHM = 10**18  # 1 SHM = 10^18 ashm (attoshm)
INITIAL_SUPPLY_SHM = 60_000_000_000  # 60 billion SHM
BLOCKS_PER_YEAR = 6_311_520  # Default blocks per year in genesis


@dataclass
class MintParams:
    """Mint module parameters"""
    mint_denom: str
    inflation_rate_change: float  # Rate at which inflation adjusts
    inflation_max: float          # Maximum inflation rate
    inflation_min: float          # Minimum inflation rate
    goal_bonded: float            # Target bonding ratio
    blocks_per_year: int          # Number of blocks per year


@dataclass
class NetworkState:
    """Current network state"""
    total_supply: float           # Total token supply (in SHM)
    bonded_tokens: float          # Currently bonded tokens (in SHM)
    current_inflation: float      # Current inflation rate


@dataclass
class SimulationResult:
    """Results from economic simulation"""
    # Inputs
    total_supply_shm: float
    bonded_tokens_shm: float
    bonded_ratio: float
    min_inflation: float
    max_inflation: float
    goal_bonded: float
    inflation_rate_change: float

    # Calculated outputs
    current_inflation: float       # Actual inflation rate after adjustment
    annual_provisions_shm: float   # New tokens minted per year (SHM)
    validator_apy: float           # APY for validators/stakers

    # Additional metrics
    inflation_delta: float         # How much inflation changed
    bonded_deviation: float        # Difference from goal bonding ratio

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            # Inputs
            'total_supply_shm': self.total_supply_shm,
            'bonded_tokens_shm': self.bonded_tokens_shm,
            'bonded_ratio': round(self.bonded_ratio, 6),
            'min_inflation': round(self.min_inflation, 6),
            'max_inflation': round(self.max_inflation, 6),
            'goal_bonded': round(self.goal_bonded, 6),
            'inflation_rate_change': round(self.inflation_rate_change, 6),

            # Outputs
            'current_inflation': round(self.current_inflation, 6),
            'annual_provisions_shm': round(self.annual_provisions_shm, 2),
            'validator_apy': round(self.validator_apy, 6),

            # Metrics
            'inflation_delta': round(self.inflation_delta, 6),
            'bonded_deviation': round(self.bonded_deviation, 6),
        }


class CosmosEconomicSimulator:
    """
    Simulates Cosmos SDK mint module inflation dynamics.

    The Cosmos mint module adjusts inflation based on the bonding ratio:
    - If bonded < goal: Increase inflation to incentivize staking
    - If bonded > goal: Decrease inflation to reduce rewards
    """

    def __init__(self, params: MintParams):
        self.params = params

    def calculate_inflation(
        self,
        current_inflation: float,
        bonded_ratio: float
    ) -> float:
        """
        Calculate adjusted inflation rate based on bonding ratio.

        Algorithm from Cosmos SDK mint module:
        1. Calculate deviation from goal bonding ratio
        2. Adjust inflation proportionally to deviation
        3. Clamp between min and max inflation

        Args:
            current_inflation: Current inflation rate (0.0 to 1.0)
            bonded_ratio: Current bonding ratio (0.0 to 1.0)

        Returns:
            Adjusted inflation rate
        """
        # Calculate deviation from goal
        bonded_deviation = self.params.goal_bonded - bonded_ratio

        # Calculate inflation adjustment
        # When below goal (positive deviation): increase inflation
        # When above goal (negative deviation): decrease inflation
        if bonded_deviation != 0:
            inflation_change = (
                self.params.inflation_rate_change *
                bonded_deviation
            )
        else:
            inflation_change = 0.0

        # Apply adjustment
        new_inflation = current_inflation + inflation_change

        # Clamp to min/max bounds
        new_inflation = max(
            self.params.inflation_min,
            min(self.params.inflation_max, new_inflation)
        )

        return new_inflation

    def calculate_annual_provisions(
        self,
        total_supply: float,
        inflation: float
    ) -> float:
        """
        Calculate annual token provisions (new tokens minted per year).

        Args:
            total_supply: Total token supply (in SHM)
            inflation: Current inflation rate (0.0 to 1.0)

        Returns:
            Annual provisions in SHM
        """
        return total_supply * inflation

    def calculate_validator_apy(
        self,
        annual_provisions: float,
        bonded_tokens: float
    ) -> float:
        """
        Calculate APY for validators/stakers.

        APY = (annual_provisions / bonded_tokens)

        Args:
            annual_provisions: New tokens minted per year (SHM)
            bonded_tokens: Total bonded tokens (SHM)

        Returns:
            Validator APY (0.0 to infinity)
        """
        if bonded_tokens <= 0:
            return 0.0

        return annual_provisions / bonded_tokens

    def simulate(
        self,
        network_state: NetworkState
    ) -> SimulationResult:
        """
        Run full economic simulation for given network state.

        Args:
            network_state: Current state of the network

        Returns:
            SimulationResult with all calculated metrics
        """
        # Calculate bonding ratio
        bonded_ratio = (
            network_state.bonded_tokens / network_state.total_supply
            if network_state.total_supply > 0
            else 0.0
        )

        # Store initial inflation for delta calculation
        initial_inflation = network_state.current_inflation

        # Calculate adjusted inflation
        new_inflation = self.calculate_inflation(
            network_state.current_inflation,
            bonded_ratio
        )

        # Calculate annual provisions
        annual_provisions = self.calculate_annual_provisions(
            network_state.total_supply,
            new_inflation
        )

        # Calculate validator APY
        validator_apy = self.calculate_validator_apy(
            annual_provisions,
            network_state.bonded_tokens
        )

        # Calculate metrics
        inflation_delta = new_inflation - initial_inflation
        bonded_deviation = self.params.goal_bonded - bonded_ratio

        return SimulationResult(
            # Inputs
            total_supply_shm=network_state.total_supply,
            bonded_tokens_shm=network_state.bonded_tokens,
            bonded_ratio=bonded_ratio,
            min_inflation=self.params.inflation_min,
            max_inflation=self.params.inflation_max,
            goal_bonded=self.params.goal_bonded,
            inflation_rate_change=self.params.inflation_rate_change,

            # Outputs
            current_inflation=new_inflation,
            annual_provisions_shm=annual_provisions,
            validator_apy=validator_apy,

            # Metrics
            inflation_delta=inflation_delta,
            bonded_deviation=bonded_deviation,
        )


def run_single_simulation(
    total_supply_shm: float = INITIAL_SUPPLY_SHM,
    bonded_ratio: float = 0.67,
    current_inflation: float = 0.13,
    min_inflation: float = 0.07,
    max_inflation: float = 0.20,
    goal_bonded: float = 0.67,
    inflation_rate_change: float = 0.13,
    blocks_per_year: int = BLOCKS_PER_YEAR,
    mint_denom: str = "ashm"
) -> SimulationResult:
    """
    Run a single simulation with specified parameters.

    Convenience function for quick simulations.

    Args:
        total_supply_shm: Total supply in SHM
        bonded_ratio: Bonding ratio (0.0 to 1.0)
        current_inflation: Starting inflation rate
        min_inflation: Minimum inflation rate
        max_inflation: Maximum inflation rate
        goal_bonded: Target bonding ratio
        inflation_rate_change: Rate of inflation adjustment
        blocks_per_year: Blocks per year
        mint_denom: Mint denomination

    Returns:
        SimulationResult with all metrics
    """
    # Create mint parameters
    params = MintParams(
        mint_denom=mint_denom,
        inflation_rate_change=inflation_rate_change,
        inflation_max=max_inflation,
        inflation_min=min_inflation,
        goal_bonded=goal_bonded,
        blocks_per_year=blocks_per_year
    )

    # Create network state
    bonded_tokens = total_supply_shm * bonded_ratio
    network_state = NetworkState(
        total_supply=total_supply_shm,
        bonded_tokens=bonded_tokens,
        current_inflation=current_inflation
    )

    # Run simulation
    simulator = CosmosEconomicSimulator(params)
    return simulator.simulate(network_state)


def main():
    """Demo: Run simulation with current network parameters"""
    print("Cosmos Economic Simulator")
    print("=" * 80)
    print()

    # Current network parameters (from genesis files)
    print("Running simulation with CURRENT network parameters:")
    print(f"  Total Supply: {INITIAL_SUPPLY_SHM:,.0f} SHM")
    print(f"  Min Inflation: 7%")
    print(f"  Max Inflation: 20%")
    print(f"  Goal Bonded: 67%")
    print(f"  Inflation Rate Change: 13%")
    print()

    # Test at different bonding ratios
    test_ratios = [0.50, 0.60, 0.67, 0.75, 0.85]

    for bonded_ratio in test_ratios:
        result = run_single_simulation(
            total_supply_shm=INITIAL_SUPPLY_SHM,
            bonded_ratio=bonded_ratio,
            current_inflation=0.13,  # Start at 13%
            min_inflation=0.07,
            max_inflation=0.20,
            goal_bonded=0.67,
            inflation_rate_change=0.13
        )

        print(f"Bonded Ratio: {bonded_ratio:.0%}")
        print(f"  Current Inflation: {result.current_inflation:.2%}")
        print(f"  Annual Provisions: {result.annual_provisions_shm:,.0f} SHM")
        print(f"  Validator APY: {result.validator_apy:.2%}")
        print(f"  New SHM/Year: {result.annual_provisions_shm:,.0f}")
        print()


if __name__ == "__main__":
    main()
