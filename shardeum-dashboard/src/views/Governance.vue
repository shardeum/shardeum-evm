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
        :key="proposal.proposalId"
        class="card p-6 hover:shadow-md transition-shadow cursor-pointer"
        @click="selectProposal(proposal)"
      >
        <div class="flex items-start justify-between">
          <div class="flex-1 pr-4">
            <div class="flex items-center space-x-3 mb-2">
              <h3 class="text-lg font-medium text-gray-900">
                #{{ proposal.proposalId }} {{ proposal.content.title }}
              </h3>
              <span
                class="inline-flex px-2 py-1 text-xs font-semibold rounded-full"
                :class="{
                  'bg-yellow-100 text-yellow-800': proposal.status === 2,
                  'bg-green-100 text-green-800': proposal.status === 3,
                  'bg-red-100 text-red-800': proposal.status === 4,
                  'bg-gray-100 text-gray-800': proposal.status === 1
                }"
              >
                {{ getStatusText(proposal.status) }}
              </span>
            </div>
            <p class="text-gray-600 mb-4 line-clamp-3">{{ proposal.content.description }}</p>
            <div class="flex items-center space-x-6 text-sm text-gray-500">
              <span>Submit Time: {{ formatDate(proposal.submitTime) }}</span>
              <span v-if="proposal.votingEndTime">Voting Ends: {{ formatDate(proposal.votingEndTime) }}</span>
            </div>
          </div>
          
          <!-- Voting Results Preview -->
          <div v-if="proposal.status === 2" class="w-32">
            <div class="text-sm font-medium text-gray-900 mb-2">Current Votes</div>
            <div class="space-y-1">
              <div class="flex justify-between text-xs">
                <span class="text-green-600">Yes</span>
                <span>{{ getVotePercentages(proposal).yes }}%</span>
              </div>
              <div class="flex justify-between text-xs">
                <span class="text-red-600">No</span>
                <span>{{ getVotePercentages(proposal).no }}%</span>
              </div>
              <div class="flex justify-between text-xs">
                <span class="text-yellow-600">Abstain</span>
                <span>{{ getVotePercentages(proposal).abstain }}%</span>
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
      @close="governanceStore.clearSelectedProposal()"
      @voted="handleVoteSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import {
  ArrowPathIcon,
  PlusIcon,
  WalletIcon,
  DocumentTextIcon
} from '@heroicons/vue/24/outline'
import { useNetworkStore } from '@/stores/network'
import { useWalletStore } from '@/stores/wallet'
import { useGovernanceStore } from '@/stores/governance'
import CreateProposalModal from '@/components/CreateProposalModal.vue'
import ProposalDetailModal from '@/components/ProposalDetailModal.vue'
import { getProposalStatusText } from '@/types/governance'

const networkStore = useNetworkStore()
const walletStore = useWalletStore()
const governanceStore = useGovernanceStore()

const showCreateModal = ref(false)

// Use governance store values
const proposals = computed(() => governanceStore.proposals)
const loading = computed(() => governanceStore.loading)
const error = computed(() => governanceStore.error)
const selectedProposal = computed(() => governanceStore.selectedProposal)

onMounted(() => {
  loadProposals()
})

async function loadProposals() {
  await governanceStore.loadProposals()
}

function selectProposal(proposal: any) {
  governanceStore.loadProposal(proposal.proposalId)
}

async function handleCreateSuccess() {
  showCreateModal.value = false
  // Immediately try to refresh, then again after a delay for blockchain processing
  await loadProposals()
  
  // Wait a moment for the transaction to be processed and try again
  setTimeout(async () => {
    await loadProposals()
  }, 3000) // Wait 3 seconds before refreshing again
}

function handleVoteSuccess() {
  loadProposals()
  // Reload the selected proposal if there is one
  if (selectedProposal.value) {
    governanceStore.loadProposal(selectedProposal.value.proposalId)
  }
}

function getStatusText(status: number): string {
  return getProposalStatusText(status)
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

function getVotePercentages(proposal: any) {
  return governanceStore.getVotePercentages(proposal)
}
</script>