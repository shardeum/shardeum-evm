import { useNetworkStore } from '@/stores/network'
import type { GovernanceProposal, ProposalVote, ProposalDeposit, GovernanceParams } from '@/types/governance'

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

class ApiService {
  private getBaseUrl(): string {
    // In development, use the Vite proxy (relative URLs)
    if (import.meta.env.DEV) {
      return ''
    }
    // In production, use the full endpoint URL
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

  async getProposals(): Promise<GovernanceProposal[]> {
    try {
      const response = await this.fetchApi('/cosmos/gov/v1beta1/proposals?pagination.limit=50')
      const proposals = response.proposals || []
      
      return proposals.map((p: any) => ({
        proposalId: p.proposal_id || p.id,
        content: {
          typeUrl: p.content?.['@type'] || '',
          title: p.content?.title || '',
          description: p.content?.description || '',
          recipient: p.content?.recipient,
          amount: p.content?.amount,
          changes: p.content?.changes
        },
        status: this.parseProposalStatus(p.status),
        finalTallyResult: {
          yes: p.final_tally_result?.yes || '0',
          abstain: p.final_tally_result?.abstain || '0',
          no: p.final_tally_result?.no || '0',
          noWithVeto: p.final_tally_result?.no_with_veto || '0'
        },
        submitTime: p.submit_time,
        depositEndTime: p.deposit_end_time,
        totalDeposit: p.total_deposit || [],
        votingStartTime: p.voting_start_time,
        votingEndTime: p.voting_end_time
      }))
    } catch (error) {
      console.error('Failed to get proposals:', error)
      return []
    }
  }

  async getProposal(proposalId: string): Promise<GovernanceProposal | null> {
    try {
      const response = await this.fetchApi(`/cosmos/gov/v1beta1/proposals/${proposalId}`)
      const p = response.proposal
      
      if (!p) return null
      
      return {
        proposalId: p.proposal_id || p.id,
        content: {
          typeUrl: p.content?.['@type'] || '',
          title: p.content?.title || '',
          description: p.content?.description || '',
          recipient: p.content?.recipient,
          amount: p.content?.amount,
          changes: p.content?.changes
        },
        status: this.parseProposalStatus(p.status),
        finalTallyResult: {
          yes: p.final_tally_result?.yes || '0',
          abstain: p.final_tally_result?.abstain || '0',
          no: p.final_tally_result?.no || '0',
          noWithVeto: p.final_tally_result?.no_with_veto || '0'
        },
        submitTime: p.submit_time,
        depositEndTime: p.deposit_end_time,
        totalDeposit: p.total_deposit || [],
        votingStartTime: p.voting_start_time,
        votingEndTime: p.voting_end_time
      }
    } catch (error) {
      console.error('Failed to get proposal:', error)
      return null
    }
  }

  async getProposalVotes(proposalId: string): Promise<ProposalVote[]> {
    try {
      const response = await this.fetchApi(`/cosmos/gov/v1beta1/proposals/${proposalId}/votes?pagination.limit=100`)
      const votes = response.votes || []
      
      return votes.map((v: any) => ({
        proposalId: v.proposal_id,
        voter: v.voter,
        option: v.option,
        options: v.options || []
      }))
    } catch (error) {
      console.error('Failed to get proposal votes:', error)
      return []
    }
  }

  async getProposalDeposits(proposalId: string): Promise<ProposalDeposit[]> {
    try {
      const response = await this.fetchApi(`/cosmos/gov/v1beta1/proposals/${proposalId}/deposits?pagination.limit=100`)
      const deposits = response.deposits || []
      
      return deposits.map((d: any) => ({
        proposalId: d.proposal_id,
        depositor: d.depositor,
        amount: d.amount || []
      }))
    } catch (error) {
      console.error('Failed to get proposal deposits:', error)
      return []
    }
  }

  async getGovernanceParams(): Promise<GovernanceParams | null> {
    try {
      // Get all params from v1 API
      const response = await this.fetchApi('/cosmos/gov/v1beta1/params/voting')
      const params = response.params || {}
      
      return {
        minDeposit: params.min_deposit || [],
        maxDepositPeriod: params.max_deposit_period || '0',
        votingPeriod: params.voting_period || '0',
        quorum: params.quorum || '0',
        threshold: params.threshold || '0',
        vetoThreshold: params.veto_threshold || '0'
      }
    } catch (error) {
      console.error('Failed to get governance params:', error)
      return null
    }
  }

  private parseProposalStatus(status: string): number {
    // Convert string status to enum number
    switch (status) {
      case 'PROPOSAL_STATUS_DEPOSIT_PERIOD':
        return 1
      case 'PROPOSAL_STATUS_VOTING_PERIOD':
        return 2
      case 'PROPOSAL_STATUS_PASSED':
        return 3
      case 'PROPOSAL_STATUS_REJECTED':
        return 4
      case 'PROPOSAL_STATUS_FAILED':
        return 5
      default:
        return 0
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