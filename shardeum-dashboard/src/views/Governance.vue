<template>
  <div class="space-y-6">
    <div class="flex justify-between items-center">
      <h1 class="text-2xl font-bold text-gray-900">Governance</h1>
      <div class="flex items-center space-x-4">
        <button
          @click="loadProposals"
          :disabled="loading"
          class="btn btn-secondary"
        >
          <ArrowPathIcon class="w-4 h-4 mr-2" :class="{ 'animate-spin': loading }" />
          Refresh
        </button>
        <button
          v-if="walletStore.isConnected"
          @click="showCreateModal = true"
          class="btn btn-primary"
        >
          <PlusIcon class="w-4 h-4 mr-2" />
          New Proposal
        </button>
      </div>
    </div>

    <!-- Connect Wallet Prompt -->
    <div v-if="!walletStore.isConnected" class="card p-8 text-center">
      <WalletIcon class="w-16 h-16 text-gray-400 mx-auto mb-4" />
      <h3 class="text-lg font-medium text-gray-900 mb-2">Connect Your Wallet</h3>
      <p class="text-gray-500 mb-6">Connect your Keplr wallet to participate in governance</p>
      <button @click="walletStore.connectKeplr" class="btn btn-primary">
        Connect Keplr Wallet
      </button>
    </div>

    <!-- Proposals List -->
    <div class="space-y-4">
      <div v-if="loading && proposals.length === 0" class="card p-8 text-center">
        <ArrowPathIcon class="w-8 h-8 text-gray-400 animate-spin mx-auto mb-4" />
        <p class="text-gray-500">Loading proposals...</p>
      </div>

      <div v-else-if="proposals.length === 0" class="card p-8 text-center">
        <DocumentTextIcon class="w-12 h-12 text-gray-400 mx-auto mb-4" />
        <h3 class="text-lg font-medium text-gray-900 mb-2">No Proposals</h3>
        <p class="text-gray-500">No governance proposals found</p>
      </div>

      <div
        v-for="proposal in proposals"
        :key="proposal.id"
        class="card p-6 hover:shadow-md transition-shadow cursor-pointer"
        @click="selectProposal(proposal)"
      >
        <div class="flex items-start justify-between">
          <div class="flex-1 pr-4">
            <div class="flex items-center space-x-3 mb-2">
              <h3 class="text-lg font-medium text-gray-900">
                #{{ proposal.id }} {{ proposal.title }}
              </h3>
              <span
                class="inline-flex px-2 py-1 text-xs font-semibold rounded-full"
                :class="{
                  'bg-yellow-100 text-yellow-800': proposal.status === 'PROPOSAL_STATUS_VOTING_PERIOD',
                  'bg-green-100 text-green-800': proposal.status === 'PROPOSAL_STATUS_PASSED',
                  'bg-red-100 text-red-800': proposal.status === 'PROPOSAL_STATUS_REJECTED',
                  'bg-gray-100 text-gray-800': proposal.status === 'PROPOSAL_STATUS_DEPOSIT_PERIOD'
                }"
              >
                {{ getStatusText(proposal.status) }}
              </span>
            </div>
            <p class="text-gray-600 mb-4 line-clamp-3">{{ proposal.description }}</p>
            <div class="flex items-center space-x-6 text-sm text-gray-500">
              <span>Submit Time: {{ formatDate(proposal.submitTime) }}</span>
              <span v-if="proposal.votingEndTime">Voting Ends: {{ formatDate(proposal.votingEndTime) }}</span>
            </div>
          </div>
          
          <!-- Voting Results Preview -->
          <div v-if="proposal.status === 'PROPOSAL_STATUS_VOTING_PERIOD'" class="w-32">
            <div class="text-sm font-medium text-gray-900 mb-2">Current Votes</div>
            <div class="space-y-1">
              <div class="flex justify-between text-xs">
                <span class="text-green-600">Yes</span>
                <span>{{ proposal.yesVotes || '0%' }}</span>
              </div>
              <div class="flex justify-between text-xs">
                <span class="text-red-600">No</span>
                <span>{{ proposal.noVotes || '0%' }}</span>
              </div>
              <div class="flex justify-between text-xs">
                <span class="text-yellow-600">Abstain</span>
                <span>{{ proposal.abstainVotes || '0%' }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create Proposal Modal -->
    <CreateProposalModal
      v-if="showCreateModal"
      @close="showCreateModal = false"
      @success="handleCreateSuccess"
    />

    <!-- Proposal Detail Modal -->
    <ProposalDetailModal
      v-if="selectedProposal"
      :proposal="selectedProposal"
      @close="selectedProposal = null"
      @voted="handleVoteSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  ArrowPathIcon,
  PlusIcon,
  WalletIcon,
  DocumentTextIcon
} from '@heroicons/vue/24/outline'
import { useNetworkStore } from '@/stores/network'
import { useWalletStore } from '@/stores/wallet'
import CreateProposalModal from '@/components/CreateProposalModal.vue'
import ProposalDetailModal from '@/components/ProposalDetailModal.vue'

interface Proposal {
  id: string
  title: string
  description: string
  status: string
  submitTime: string
  votingStartTime?: string
  votingEndTime?: string
  yesVotes?: string
  noVotes?: string
  abstainVotes?: string
  noWithVetoVotes?: string
}

const networkStore = useNetworkStore()
const walletStore = useWalletStore()

const proposals = ref<Proposal[]>([])
const loading = ref(false)
const showCreateModal = ref(false)
const selectedProposal = ref<Proposal | null>(null)

onMounted(() => {
  loadProposals()
})

async function loadProposals() {
  loading.value = true
  try {
    // For now, we'll show some mock data since governance queries can be complex
    // In a real implementation, you would query the governance module
    await new Promise(resolve => setTimeout(resolve, 1000)) // Simulate API call
    
    proposals.value = [
      {
        id: '1',
        title: 'Community Pool Spend - Network Upgrade',
        description: 'Proposal to fund network upgrade development from the community pool. This will ensure our network stays up to date with the latest features and security improvements.',
        status: 'PROPOSAL_STATUS_VOTING_PERIOD',
        submitTime: new Date().toISOString(),
        votingEndTime: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
        yesVotes: '67.5%',
        noVotes: '15.2%',
        abstainVotes: '17.3%'
      },
      {
        id: '2',
        title: 'Parameter Change - Minimum Gas Price',
        description: 'Proposal to adjust the minimum gas price to improve network performance and reduce spam transactions.',
        status: 'PROPOSAL_STATUS_PASSED',
        submitTime: new Date(Date.now() - 10 * 24 * 60 * 60 * 1000).toISOString(),
        yesVotes: '89.2%',
        noVotes: '10.8%',
        abstainVotes: '0%'
      }
    ]
  } catch (error) {
    console.error('Failed to load proposals:', error)
  } finally {
    loading.value = false
  }
}

function selectProposal(proposal: Proposal) {
  selectedProposal.value = proposal
}

function handleCreateSuccess() {
  showCreateModal.value = false
  loadProposals()
}

function handleVoteSuccess() {
  loadProposals()
}

function getStatusText(status: string): string {
  switch (status) {
    case 'PROPOSAL_STATUS_DEPOSIT_PERIOD':
      return 'Deposit Period'
    case 'PROPOSAL_STATUS_VOTING_PERIOD':
      return 'Voting'
    case 'PROPOSAL_STATUS_PASSED':
      return 'Passed'
    case 'PROPOSAL_STATUS_REJECTED':
      return 'Rejected'
    case 'PROPOSAL_STATUS_FAILED':
      return 'Failed'
    default:
      return 'Unknown'
  }
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>