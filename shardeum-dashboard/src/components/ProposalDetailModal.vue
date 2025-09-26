<template>
  <div class="fixed inset-0 z-50 overflow-y-auto">
    <div class="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0">
      <div class="fixed inset-0 transition-opacity bg-gray-500 bg-opacity-75" @click="$emit('close')"></div>

      <div class="inline-block w-full max-w-4xl p-6 my-8 overflow-hidden text-left align-middle transition-all transform bg-white shadow-xl rounded-lg">
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center space-x-3">
            <h3 class="text-xl font-medium text-gray-900">
              Proposal #{{ proposal.proposalId }}
            </h3>
            <span
              class="inline-flex px-3 py-1 text-sm font-semibold rounded-full"
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
          <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-6 h-6" />
          </button>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <!-- Main Content -->
          <div class="lg:col-span-2 space-y-6">
            <!-- Title and Description -->
            <div>
              <h4 class="text-lg font-medium text-gray-900 mb-3">{{ proposal.content.title }}</h4>
              <div class="prose max-w-none text-gray-700">
                <p class="whitespace-pre-wrap">{{ proposal.content.description }}</p>
              </div>
            </div>

            <!-- Timeline -->
            <div class="border-t pt-6">
              <h5 class="text-base font-medium text-gray-900 mb-4">Timeline</h5>
              <div class="space-y-3">
                <div class="flex items-center">
                  <div class="w-2 h-2 bg-blue-500 rounded-full mr-3"></div>
                  <div class="text-sm">
                    <span class="font-medium">Submitted:</span>
                    <span class="text-gray-600 ml-1">{{ formatDate(proposal.submitTime) }}</span>
                  </div>
                </div>
                <div v-if="proposal.votingStartTime" class="flex items-center">
                  <div class="w-2 h-2 bg-green-500 rounded-full mr-3"></div>
                  <div class="text-sm">
                    <span class="font-medium">Voting Started:</span>
                    <span class="text-gray-600 ml-1">{{ formatDate(proposal.votingStartTime) }}</span>
                  </div>
                </div>
                <div v-if="proposal.votingEndTime" class="flex items-center">
                  <div 
                    class="w-2 h-2 rounded-full mr-3"
                    :class="isVotingActive ? 'bg-yellow-500' : 'bg-gray-500'"
                  ></div>
                  <div class="text-sm">
                    <span class="font-medium">Voting {{ isVotingActive ? 'Ends' : 'Ended' }}:</span>
                    <span class="text-gray-600 ml-1">{{ formatDate(proposal.votingEndTime) }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Sidebar -->
          <div class="space-y-6">
            <!-- Voting Results -->
            <div v-if="proposal.status === 2" class="card p-4">
              <h5 class="text-base font-medium text-gray-900 mb-4">Current Results</h5>
              <div class="space-y-3">
                <div class="flex justify-between items-center">
                  <span class="text-sm text-green-600 font-medium">Yes</span>
                  <span class="text-sm font-medium">{{ getVotePercentages(proposal).yes }}%</span>
                </div>
                <div class="w-full bg-gray-200 rounded-full h-2">
                  <div 
                    class="bg-green-500 h-2 rounded-full" 
                    :style="`width: ${getVotePercentages(proposal).yes}%`"
                  ></div>
                </div>
                
                <div class="flex justify-between items-center">
                  <span class="text-sm text-red-600 font-medium">No</span>
                  <span class="text-sm font-medium">{{ getVotePercentages(proposal).no }}%</span>
                </div>
                <div class="w-full bg-gray-200 rounded-full h-2">
                  <div 
                    class="bg-red-500 h-2 rounded-full" 
                    :style="`width: ${getVotePercentages(proposal).no}%`"
                  ></div>
                </div>
                
                <div class="flex justify-between items-center">
                  <span class="text-sm text-yellow-600 font-medium">Abstain</span>
                  <span class="text-sm font-medium">{{ getVotePercentages(proposal).abstain }}%</span>
                </div>
                <div class="w-full bg-gray-200 rounded-full h-2">
                  <div 
                    class="bg-yellow-500 h-2 rounded-full" 
                    :style="`width: ${getVotePercentages(proposal).abstain}%`"
                  ></div>
                </div>

                <div class="flex justify-between items-center">
                  <span class="text-sm text-purple-600 font-medium">No with Veto</span>
                  <span class="text-sm font-medium">{{ getVotePercentages(proposal).noWithVeto }}%</span>
                </div>
                <div class="w-full bg-gray-200 rounded-full h-2">
                  <div 
                    class="bg-purple-500 h-2 rounded-full" 
                    :style="`width: ${getVotePercentages(proposal).noWithVeto}%`"
                  ></div>
                </div>
              </div>
            </div>

            <!-- Vote Buttons -->
            <div v-if="walletStore.isConnected && isVotingActive" class="card p-4">
              <h5 class="text-base font-medium text-gray-900 mb-4">Cast Your Vote</h5>
              <div class="space-y-2">
                <button
                  @click="vote('yes')"
                  :disabled="voting"
                  class="w-full btn text-white bg-green-600 hover:bg-green-700"
                >
                  <CheckIcon class="w-4 h-4 mr-2" />
                  Vote Yes
                </button>
                <button
                  @click="vote('no')"
                  :disabled="voting"
                  class="w-full btn text-white bg-red-600 hover:bg-red-700"
                >
                  <XMarkIcon class="w-4 h-4 mr-2" />
                  Vote No
                </button>
                <button
                  @click="vote('abstain')"
                  :disabled="voting"
                  class="w-full btn text-white bg-yellow-600 hover:bg-yellow-700"
                >
                  <MinusIcon class="w-4 h-4 mr-2" />
                  Abstain
                </button>
                <button
                  @click="vote('no_with_veto')"
                  :disabled="voting"
                  class="w-full btn text-white bg-purple-600 hover:bg-purple-700"
                >
                  <ExclamationTriangleIcon class="w-4 h-4 mr-2" />
                  No with Veto
                </button>
              </div>
              
              <div v-if="voting" class="mt-4 text-center">
                <ArrowPathIcon class="w-5 h-5 animate-spin mx-auto text-gray-400" />
                <p class="text-sm text-gray-500 mt-2">Submitting vote...</p>
              </div>
            </div>

            <!-- Connect Wallet -->
            <div v-else-if="!walletStore.isConnected && isVotingActive" class="card p-4 text-center">
              <WalletIcon class="w-8 h-8 text-gray-400 mx-auto mb-3" />
              <p class="text-sm text-gray-600 mb-4">Connect your wallet to vote</p>
              <button @click="walletStore.connectKeplr" class="btn btn-primary">
                Connect Wallet
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  XMarkIcon,
  ArrowPathIcon,
  WalletIcon,
  CheckIcon,
  MinusIcon,
  ExclamationTriangleIcon
} from '@heroicons/vue/24/outline'
import { useWalletStore } from '@/stores/wallet'
import { useGovernanceStore } from '@/stores/governance'
import type { GovernanceProposal, VoteOptionString } from '@/types/governance'
import { getProposalStatusText, calculateVotePercentages } from '@/types/governance'

const props = defineProps<{
  proposal: GovernanceProposal
}>()

const emit = defineEmits<{
  close: []
  voted: []
}>()

const walletStore = useWalletStore()
const governanceStore = useGovernanceStore()
const voting = ref(false)

const isVotingActive = computed(() => {
  return governanceStore.isVotingActive(props.proposal)
})

function getVotePercentages(proposal: any) {
  return governanceStore.getVotePercentages(proposal)
}

async function vote(option: VoteOptionString) {
  if (!walletStore.isConnected) return
  
  voting.value = true
  try {
    await governanceStore.voteOnProposal(props.proposal.proposalId, option)
    
    alert(`Vote "${option}" submitted successfully!`)
    emit('voted')
    
  } catch (error) {
    console.error('Failed to vote:', error)
    alert('Failed to submit vote. Please try again.')
  } finally {
    voting.value = false
  }
}

function getStatusText(status: number): string {
  return getProposalStatusText(status)
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>