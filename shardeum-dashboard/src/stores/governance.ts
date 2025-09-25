import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useWalletStore } from './wallet'
import { useNetworkStore } from './network'
import { apiService } from '@/services/api'
import type { 
  GovernanceProposal, 
  ProposalVote, 
  ProposalDeposit, 
  GovernanceParams,
  VoteOptionString
} from '@/types/governance'
import { calculateVotePercentages, voteOptionStringToEnum } from '@/types/governance'

export const useGovernanceStore = defineStore('governance', () => {
  const walletStore = useWalletStore()
  const networkStore = useNetworkStore()
  
  const proposals = ref<GovernanceProposal[]>([])
  const selectedProposal = ref<GovernanceProposal | null>(null)
  const proposalVotes = ref<Map<string, ProposalVote[]>>(new Map())
  const proposalDeposits = ref<Map<string, ProposalDeposit[]>>(new Map())
  const governanceParams = ref<GovernanceParams | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  
  const activeProposals = computed(() => {
    return proposals.value.filter(p => 
      p.status === 1 || p.status === 2 // DEPOSIT_PERIOD or VOTING_PERIOD
    )
  })
  
  const completedProposals = computed(() => {
    return proposals.value.filter(p => 
      p.status === 3 || p.status === 4 || p.status === 5 // PASSED, REJECTED, FAILED
    )
  })
  
  async function loadProposals() {
    loading.value = true
    error.value = null
    
    try {
      const data = await apiService.getProposals()
      proposals.value = data.sort((a, b) => 
        parseInt(b.proposalId) - parseInt(a.proposalId)
      )
    } catch (err) {
      error.value = 'Failed to load proposals'
      console.error('Failed to load proposals:', err)
    } finally {
      loading.value = false
    }
  }

  async function loadProposal(proposalId: string) {
    loading.value = true
    error.value = null
    
    try {
      const proposal = await apiService.getProposal(proposalId)
      if (proposal) {
        selectedProposal.value = proposal
        
        // Also load votes and deposits for this proposal
        await Promise.all([
          loadProposalVotes(proposalId),
          loadProposalDeposits(proposalId)
        ])
      }
    } catch (err) {
      error.value = 'Failed to load proposal'
      console.error('Failed to load proposal:', err)
    } finally {
      loading.value = false
    }
  }

  async function loadProposalVotes(proposalId: string) {
    try {
      const votes = await apiService.getProposalVotes(proposalId)
      proposalVotes.value.set(proposalId, votes)
    } catch (err) {
      console.error('Failed to load proposal votes:', err)
    }
  }

  async function loadProposalDeposits(proposalId: string) {
    try {
      const deposits = await apiService.getProposalDeposits(proposalId)
      proposalDeposits.value.set(proposalId, deposits)
    } catch (err) {
      console.error('Failed to load proposal deposits:', err)
    }
  }

  async function loadGovernanceParams() {
    try {
      const params = await apiService.getGovernanceParams()
      governanceParams.value = params
    } catch (err) {
      console.error('Failed to load governance params:', err)
    }
  }
  
  function getProposalVotes(proposalId: string): ProposalVote[] {
    return proposalVotes.value.get(proposalId) || []
  }

  function getProposalDeposits(proposalId: string): ProposalDeposit[] {
    return proposalDeposits.value.get(proposalId) || []
  }

  function getVotePercentages(proposal: GovernanceProposal) {
    return calculateVotePercentages(proposal.finalTallyResult)
  }

  function isVotingActive(proposal: GovernanceProposal): boolean {
    return proposal.status === 2 && // VOTING_PERIOD
           proposal.votingEndTime && 
           new Date(proposal.votingEndTime) > new Date()
  }

  function isDepositPeriod(proposal: GovernanceProposal): boolean {
    return proposal.status === 1 && // DEPOSIT_PERIOD
           proposal.depositEndTime && 
           new Date(proposal.depositEndTime) > new Date()
  }

  function formatProposalType(typeUrl: string): string {
    switch (typeUrl) {
      case '/cosmos.gov.v1beta1.TextProposal':
        return 'Text Proposal'
      case '/cosmos.distribution.v1beta1.CommunityPoolSpendProposal':
        return 'Community Spend'
      case '/cosmos.params.v1beta1.ParameterChangeProposal':
        return 'Parameter Change'
      case '/cosmos.upgrade.v1beta1.SoftwareUpgradeProposal':
        return 'Software Upgrade'
      case '/cosmos.upgrade.v1beta1.CancelSoftwareUpgradeProposal':
        return 'Cancel Upgrade'
      default:
        return 'Unknown'
    }
  }

  // Transaction functions using ping.pub approach
  async function voteOnProposal(proposalId: string, option: VoteOptionString) {
    if (!walletStore.isConnected) {
      throw new Error('Wallet not connected')
    }
    
    const voteOptionValue = voteOptionStringToEnum(option)
    return await walletStore.voteOnProposalManually(proposalId, voteOptionValue)
  }

  async function submitProposal(
    proposalType: 'text' | 'community-spend' | 'param-change' | 'software-upgrade' | 'cancel-upgrade',
    title: string,
    description: string,
    initialDeposit: string,
    // For community spend
    recipient?: string,
    spendAmount?: string,
    // For parameter change
    paramSubspace?: string,
    paramKey?: string,
    paramValue?: string,
    // For software upgrade
    upgradeName?: string,
    upgradeHeight?: number,
    upgradeInfo?: string,
    // For cancel upgrade
    cancelUpgradeName?: string
  ) {
    if (!walletStore.isConnected) {
      throw new Error('Wallet not connected')
    }
    
    const { currentNetwork } = networkStore
    
    return await walletStore.submitProposalManually(
      proposalType,
      title,
      description,
      initialDeposit,
      currentNetwork.baseDenom,
      recipient,
      spendAmount,
      paramSubspace,
      paramKey,
      paramValue,
      upgradeName,
      upgradeHeight,
      upgradeInfo,
      cancelUpgradeName
    )
  }

  function clearSelectedProposal() {
    selectedProposal.value = null
  }

  // Initialize on store creation
  loadGovernanceParams()
  
  return {
    proposals,
    selectedProposal,
    governanceParams,
    loading,
    error,
    activeProposals,
    completedProposals,
    loadProposals,
    loadProposal,
    loadProposalVotes,
    loadProposalDeposits,
    loadGovernanceParams,
    getProposalVotes,
    getProposalDeposits,
    getVotePercentages,
    isVotingActive,
    isDepositPeriod,
    formatProposalType,
    voteOnProposal,
    submitProposal,
    clearSelectedProposal
  }
})