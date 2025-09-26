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
      
      // Load votes for all proposals to show vote breakdown
      await Promise.all(
        proposals.value.map(proposal => loadProposalVotes(proposal.proposalId))
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
    // Always try to calculate from individual votes first (for better breakdown)
    const votes = getProposalVotes(proposal.proposalId)
    if (votes.length > 0) {
      return calculateLiveVotePercentages(votes)
    }
    
    // Fallback to final tally if no individual votes are loaded
    return calculateVotePercentages(proposal.finalTallyResult)
  }
  
  function calculateLiveVotePercentages(votes: ProposalVote[]) {
    const tallies = {
      yes: 0,
      no: 0,
      abstain: 0,
      noWithVeto: 0
    }
    
    votes.forEach(vote => {
      switch (vote.option) {
        case 1: // VOTE_OPTION_YES
          tallies.yes++
          break
        case 2: // VOTE_OPTION_ABSTAIN
          tallies.abstain++
          break
        case 3: // VOTE_OPTION_NO
          tallies.no++
          break
        case 4: // VOTE_OPTION_NO_WITH_VETO
          tallies.noWithVeto++
          break
      }
    })
    
    const total = tallies.yes + tallies.no + tallies.abstain + tallies.noWithVeto
    
    if (total === 0) {
      return { yes: 0, no: 0, abstain: 0, noWithVeto: 0, total: 0n }
    }
    
    return {
      yes: Math.round((tallies.yes / total) * 100),
      no: Math.round((tallies.no / total) * 100),
      abstain: Math.round((tallies.abstain / total) * 100),
      noWithVeto: Math.round((tallies.noWithVeto / total) * 100),
      total: BigInt(total)
    }
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

  function getVoteOptionText(option: number): string {
    switch (option) {
      case 1:
        return 'Yes'
      case 2:
        return 'Abstain'
      case 3:
        return 'No'
      case 4:
        return 'No with Veto'
      default:
        return 'Unknown'
    }
  }

  function getIndividualVotes(proposalId: string) {
    const votes = getProposalVotes(proposalId)
    return votes.map(vote => ({
      voter: vote.voter,
      option: vote.option,
      optionText: getVoteOptionText(vote.option),
      shortVoter: vote.voter.substring(0, 10) + '...' + vote.voter.substring(vote.voter.length - 6)
    }))
  }

  // Transaction functions using ping.pub approach
  async function voteOnProposal(proposalId: string, option: VoteOptionString) {
    if (!walletStore.isConnected) {
      throw new Error('Wallet not connected')
    }
    
    console.log('Voting on proposal:', { proposalId, option, walletAddress: walletStore.address })
    
    const voteOptionValue = voteOptionStringToEnum(option)
    return await walletStore.voteOnProposalManually(proposalId, voteOptionValue)
  }

  // Proposal submission removed - focusing on voting functionality only
  // Proposals should be submitted through governance forums or by validators

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
    getVoteOptionText,
    getIndividualVotes,
    voteOnProposal,
    // submitProposal, // Removed - focusing on voting only
    clearSelectedProposal
  }
})