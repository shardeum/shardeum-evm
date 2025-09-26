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
        <!-- Proposal creation removed - focus on voting functionality -->
        <div v-if="walletStore.isConnected" class="text-sm text-gray-600">
          Connected: {{ walletStore.shortAddress }}
        </div>
      </div>
    </div>

    <!-- Connect Wallet Prompt -->
    <div v-if="!walletStore.isConnected" class="card p-8 text-center">
      <WalletIcon class="w-16 h-16 text-gray-400 mx-auto mb-4" />
      <h3 class="text-lg font-medium text-gray-900 mb-2">Connect Your Wallet</h3>
      <p class="text-gray-500 mb-6">Connect your Keplr wallet to vote on governance proposals</p>
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
        <p class="text-gray-500 mb-4">No governance proposals found.</p>
        <div class="bg-blue-50 border border-blue-200 rounded-lg p-4 text-left max-w-md mx-auto">
          <div class="flex">
            <InformationCircleIcon class="w-5 h-5 text-blue-400 mr-2 mt-0.5 flex-shrink-0" />
            <div class="text-sm text-blue-800">
              <div class="font-medium mb-1">How are proposals created?</div>
              <div>Proposals are typically submitted by validators or core developers using the CLI or governance forums. This dashboard focuses on voting functionality.</div>
            </div>
          </div>
        </div>
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

    <!-- Proposal creation removed - focusing on voting functionality -->

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
  DocumentTextIcon,
  InformationCircleIcon
} from '@heroicons/vue/24/outline'
import { useNetworkStore } from '@/stores/network'
import { useWalletStore } from '@/stores/wallet'
import { useGovernanceStore } from '@/stores/governance'
// import CreateProposalModal from '@/components/CreateProposalModal.vue' // Removed - focusing on voting only
import ProposalDetailModal from '@/components/ProposalDetailModal.vue'
import { getProposalStatusText } from '@/types/governance'

const networkStore = useNetworkStore()
const walletStore = useWalletStore()
const governanceStore = useGovernanceStore()

// const showCreateModal = ref(false) // Removed - focusing on voting only

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

// Proposal creation functions removed - focusing on voting only

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