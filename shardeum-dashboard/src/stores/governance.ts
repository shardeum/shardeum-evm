import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useWalletStore } from './wallet'
import { apiService, type Proposal } from '@/services/api'

export const useGovernanceStore = defineStore('governance', () => {
  const walletStore = useWalletStore()
  
  const proposals = ref<Proposal[]>([])
  const isLoading = ref(false)
  
  const activeProposals = computed(() => {
    return proposals.value.filter(p => 
      p.status === 'PROPOSAL_STATUS_VOTING_PERIOD' || 
      p.status === 'PROPOSAL_STATUS_DEPOSIT_PERIOD'
    )
  })
  
  const passedProposals = computed(() => {
    return proposals.value.filter(p => p.status === 'PROPOSAL_STATUS_PASSED')
  })
  
  const rejectedProposals = computed(() => {
    return proposals.value.filter(p => 
      p.status === 'PROPOSAL_STATUS_REJECTED' ||
      p.status === 'PROPOSAL_STATUS_FAILED'
    )
  })
  
  async function loadProposals() {
    try {
      isLoading.value = true
      proposals.value = await apiService.getProposals()
    } catch (error) {
      console.error('Failed to load proposals:', error)
    } finally {
      isLoading.value = false
    }
  }
  
  async function getProposal(proposalId: string): Promise<Proposal | null> {
    try {
      return await apiService.getProposal(proposalId)
    } catch (error) {
      console.error('Failed to get proposal:', error)
      return null
    }
  }
  
  async function getProposalVotes(proposalId: string): Promise<any[]> {
    try {
      return await apiService.getProposalVotes(proposalId)
    } catch (error) {
      console.error('Failed to get proposal votes:', error)
      return []
    }
  }
  
  // Transaction functions would require the signing client
  async function voteOnProposal(proposalId: string, vote: 'VOTE_OPTION_YES' | 'VOTE_OPTION_NO' | 'VOTE_OPTION_ABSTAIN' | 'VOTE_OPTION_NO_WITH_VETO') {
    const signingClient = await walletStore.getSigningClient()
    if (!signingClient || !walletStore.address) {
      throw new Error('Wallet not connected')
    }
    
    // This would implement the actual vote transaction
    console.log('Vote on proposal:', { proposalId, vote })
    // TODO: Implement vote transaction
  }
  
  async function depositToProposal(proposalId: string, amount: string) {
    const signingClient = await walletStore.getSigningClient()
    if (!signingClient || !walletStore.address) {
      throw new Error('Wallet not connected')
    }
    
    // This would implement the actual deposit transaction
    console.log('Deposit to proposal:', { proposalId, amount })
    // TODO: Implement deposit transaction
  }
  
  async function submitProposal(content: any, initialDeposit: string) {
    const signingClient = await walletStore.getSigningClient()
    if (!signingClient || !walletStore.address) {
      throw new Error('Wallet not connected')
    }
    
    // This would implement the actual proposal submission
    console.log('Submit proposal:', { content, initialDeposit })
    // TODO: Implement proposal submission
  }
  
  return {
    proposals,
    activeProposals,
    passedProposals,
    rejectedProposals,
    isLoading,
    loadProposals,
    getProposal,
    getProposalVotes,
    voteOnProposal,
    depositToProposal,
    submitProposal
  }
})