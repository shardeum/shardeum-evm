import { useNetworkStore } from '@/stores/network'

export interface ChainInfo {
  chainId: string
  nodeInfo: {
    id: string
    version: string
    moniker: string
  }
  syncInfo: {
    latestBlockHeight: string
    latestBlockTime: string
    catchingUp: boolean
  }
}

export interface Account {
  address: string
  sequence: string
  accountNumber: string
}

export interface Balance {
  denom: string
  amount: string
}

export interface Validator {
  operatorAddress: string
  consensusPubkey: any
  jailed: boolean
  status: string
  tokens: string
  delegatorShares: string
  description: {
    moniker: string
    identity: string
    website: string
    details: string
  }
  commission: {
    commissionRates: {
      rate: string
      maxRate: string
      maxChangeRate: string
    }
  }
}

export interface Proposal {
  proposalId: string
  content: any
  status: string
  finalTallyResult: {
    yes: string
    abstain: string
    no: string
    noWithVeto: string
  }
  submitTime: string
  depositEndTime: string
  totalDeposit: Balance[]
  votingStartTime: string
  votingEndTime: string
}

class ApiService {
  private getBaseUrl(): string {
    const networkStore = useNetworkStore()
    return networkStore.currentNetwork.apiEndpoint
  }

  private async fetchApi(endpoint: string): Promise<any> {
    const baseUrl = this.getBaseUrl()
    const url = `${baseUrl}${endpoint}`
    
    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      return await response.json()
    } catch (error) {
      console.error(`API call failed for ${endpoint}:`, error)
      throw error
    }
  }

  async getNodeInfo(): Promise<ChainInfo> {
    const response = await this.fetchApi('/cosmos/base/tendermint/v1beta1/node_info')
    return {
      chainId: response.default_node_info?.network || '',
      nodeInfo: {
        id: response.default_node_info?.default_node_id || '',
        version: response.default_node_info?.version || '',
        moniker: response.default_node_info?.moniker || ''
      },
      syncInfo: {
        latestBlockHeight: '1', // We'll get this from blocks endpoint separately  
        latestBlockTime: new Date().toISOString(),
        catchingUp: false
      }
    }
  }

  async getAccount(address: string): Promise<Account | null> {
    try {
      const response = await this.fetchApi(`/cosmos/auth/v1beta1/accounts/${address}`)
      const account = response.account
      
      return {
        address: account.address,
        sequence: account.sequence || '0',
        accountNumber: account.account_number || '0'
      }
    } catch (error) {
      console.warn('Account not found:', address)
      return null
    }
  }

  async getBalance(address: string, denom?: string): Promise<Balance[]> {
    try {
      const response = await this.fetchApi(`/cosmos/bank/v1beta1/balances/${address}`)
      let balances = response.balances || []
      
      if (denom) {
        balances = balances.filter((b: Balance) => b.denom === denom)
      }
      
      return balances
    } catch (error) {
      console.error('Failed to get balance:', error)
      return []
    }
  }

  async getValidators(): Promise<Validator[]> {
    try {
      const response = await this.fetchApi('/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED&pagination.limit=100')
      const validators = response.validators || []
      
      // Map API response to our interface
      return validators.map((v: any) => ({
        operatorAddress: v.operator_address,
        consensusPubkey: v.consensus_pubkey,
        jailed: v.jailed,
        status: v.status,
        tokens: v.tokens,
        delegatorShares: v.delegator_shares,
        description: {
          moniker: v.description.moniker,
          identity: v.description.identity,
          website: v.description.website,
          details: v.description.details
        },
        commission: {
          commissionRates: {
            rate: v.commission.commission_rates.rate,
            maxRate: v.commission.commission_rates.max_rate,
            maxChangeRate: v.commission.commission_rates.max_change_rate
          }
        }
      }))
    } catch (error) {
      console.error('Failed to get validators:', error)
      return []
    }
  }

  async getDelegations(delegatorAddress: string): Promise<any[]> {
    try {
      const response = await this.fetchApi(`/cosmos/staking/v1beta1/delegations/${delegatorAddress}`)
      return response.delegation_responses || []
    } catch (error) {
      console.error('Failed to get delegations:', error)
      return []
    }
  }

  async getUnbondingDelegations(delegatorAddress: string): Promise<any[]> {
    try {
      const response = await this.fetchApi(`/cosmos/staking/v1beta1/delegators/${delegatorAddress}/unbonding_delegations`)
      return response.unbonding_responses || []
    } catch (error) {
      console.error('Failed to get unbonding delegations:', error)
      return []
    }
  }

  async getRewards(delegatorAddress: string): Promise<any> {
    try {
      const response = await this.fetchApi(`/cosmos/distribution/v1beta1/delegators/${delegatorAddress}/rewards`)
      return response || { rewards: [], total: [] }
    } catch (error) {
      console.error('Failed to get rewards:', error)
      return { rewards: [], total: [] }
    }
  }

  async getProposals(): Promise<Proposal[]> {
    try {
      const response = await this.fetchApi('/cosmos/gov/v1beta1/proposals?pagination.limit=50')
      return response.proposals || []
    } catch (error) {
      console.error('Failed to get proposals:', error)
      return []
    }
  }

  async getProposal(proposalId: string): Promise<Proposal | null> {
    try {
      const response = await this.fetchApi(`/cosmos/gov/v1beta1/proposals/${proposalId}`)
      return response.proposal || null
    } catch (error) {
      console.error('Failed to get proposal:', error)
      return null
    }
  }

  async getProposalVotes(proposalId: string): Promise<any[]> {
    try {
      const response = await this.fetchApi(`/cosmos/gov/v1beta1/proposals/${proposalId}/votes`)
      return response.votes || []
    } catch (error) {
      console.error('Failed to get proposal votes:', error)
      return []
    }
  }

  async getLatestBlocks(limit: number = 20): Promise<any[]> {
    try {
      const response = await this.fetchApi(`/cosmos/base/tendermint/v1beta1/blocks/latest`)
      const latestHeight = parseInt(response.block?.header?.height || '0')
      
      // Get recent blocks by fetching multiple block heights
      const blocks = []
      for (let i = 0; i < Math.min(limit, 10); i++) {
        try {
          const blockResponse = await this.fetchApi(`/cosmos/base/tendermint/v1beta1/blocks/${latestHeight - i}`)
          blocks.push(blockResponse.block)
        } catch (error) {
          break // Stop if we can't fetch a block
        }
      }
      
      return blocks
    } catch (error) {
      console.error('Failed to get latest blocks:', error)
      return []
    }
  }

  async getBlockByHeight(height: number): Promise<any | null> {
    try {
      const response = await this.fetchApi(`/cosmos/base/tendermint/v1beta1/blocks/${height}`)
      return response.block
    } catch (error) {
      console.error(`Failed to get block ${height}:`, error)
      return null
    }
  }
}

export const apiService = new ApiService()