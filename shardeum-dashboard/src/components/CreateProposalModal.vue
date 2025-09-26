<template>
  <div class="fixed inset-0 z-50 overflow-y-auto">
    <div class="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0">
      <div class="fixed inset-0 transition-opacity bg-gray-500 bg-opacity-75" @click="$emit('close')"></div>

      <div class="inline-block w-full max-w-2xl p-6 my-8 overflow-hidden text-left align-middle transition-all transform bg-white shadow-xl rounded-lg">
        <div class="flex items-center justify-between mb-6">
          <h3 class="text-lg font-medium text-gray-900">Create New Proposal</h3>
          <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-6 h-6" />
          </button>
        </div>

        <form @submit.prevent="submitProposal" class="space-y-6">
          <!-- Proposal Type -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">
              Proposal Type
            </label>
            <select v-model="proposalType" class="input">
              <option value="text">Text Proposal (Signaling)</option>
              <!-- Other proposal types temporarily disabled for cosmos.gov.v1 migration -->
              <!-- <option value="community-spend">Community Pool Spend</option>
              <option value="param-change">Parameter Change</option>
              <option value="software-upgrade">Software Upgrade</option>
              <option value="cancel-upgrade">Cancel Software Upgrade</option> -->
            </select>
            <p class="text-xs text-gray-500 mt-1">
              Note: Only text proposals are currently supported. Other proposal types coming soon.
            </p>
          </div>

          <!-- Title -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">
              Title *
            </label>
            <input
              v-model="title"
              type="text"
              required
              class="input"
              placeholder="Enter proposal title"
            />
          </div>

          <!-- Description -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">
              Description *
            </label>
            <textarea
              v-model="description"
              required
              rows="6"
              class="input"
              placeholder="Detailed description of the proposal"
            />
          </div>

          <!-- Initial Deposit -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">
              Initial Deposit
            </label>
            <div class="relative">
              <input
                v-model="deposit"
                type="number"
                step="0.000001"
                class="input pr-20"
                placeholder="0.0"
              />
              <div class="absolute inset-y-0 right-0 flex items-center pr-3">
                <span class="text-sm text-gray-500">{{ networkStore.currentNetwork.symbol }}</span>
              </div>
            </div>
            <p class="text-sm text-gray-500 mt-1">
              Minimum deposit required: 10 {{ networkStore.currentNetwork.symbol }}
            </p>
          </div>

          <!-- Community Spend Specific Fields -->
          <div v-if="proposalType === 'community-spend'" class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">
                Recipient Address *
              </label>
              <input
                v-model="recipientAddress"
                type="text"
                class="input font-mono"
                placeholder="shardeum1..."
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">
                Amount to Spend
              </label>
              <div class="relative">
                <input
                  v-model="spendAmount"
                  type="number"
                  step="0.000001"
                  class="input pr-20"
                  placeholder="0.0"
                />
                <div class="absolute inset-y-0 right-0 flex items-center pr-3">
                  <span class="text-sm text-gray-500">{{ networkStore.currentNetwork.symbol }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Parameter Change Specific Fields -->
          <div v-if="proposalType === 'param-change'" class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">
                Parameter Subspace *
              </label>
              <input
                v-model="paramSubspace"
                type="text"
                class="input"
                placeholder="e.g., staking, gov, mint"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">
                Parameter Key *
              </label>
              <input
                v-model="paramKey"
                type="text"
                class="input"
                placeholder="e.g., MaxValidators, DepositParams"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">
                New Value *
              </label>
              <input
                v-model="paramValue"
                type="text"
                class="input"
                placeholder="Enter the new parameter value"
              />
            </div>
          </div>

          <!-- Software Upgrade Specific Fields -->
          <div v-if="proposalType === 'software-upgrade'" class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">
                Upgrade Name *
              </label>
              <input
                v-model="upgradeName"
                type="text"
                class="input"
                placeholder="e.g., v2.0.0"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">
                Upgrade Height *
              </label>
              <input
                v-model="upgradeHeight"
                type="number"
                class="input"
                placeholder="Block height for upgrade"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">
                Upgrade Info
              </label>
              <textarea
                v-model="upgradeInfo"
                rows="3"
                class="input"
                placeholder="JSON upgrade info (optional)"
              />
            </div>
          </div>

          <!-- Cancel Software Upgrade Specific Fields -->
          <div v-if="proposalType === 'cancel-upgrade'" class="bg-blue-50 border border-blue-200 rounded-lg p-4">
            <div class="flex">
              <InformationCircleIcon class="w-5 h-5 text-blue-400 mr-2 mt-0.5" />
              <div class="text-sm text-blue-800">
                <div class="font-medium mb-1">Cancel Upgrade Proposal</div>
                <div>This proposal will cancel any currently scheduled software upgrade. No additional parameters are required.</div>
              </div>
            </div>
          </div>

          <!-- Warning -->
          <div class="p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
            <div class="flex">
              <ExclamationTriangleIcon class="w-5 h-5 text-yellow-400 mr-2 mt-0.5" />
              <div class="text-sm text-yellow-800">
                <div class="font-medium mb-1">Important</div>
                <div>Creating a proposal requires a deposit and careful consideration. Make sure all details are correct before submitting.</div>
              </div>
            </div>
          </div>

          <!-- Action Buttons -->
          <div class="flex space-x-3 pt-4">
            <button
              type="button"
              @click="$emit('close')"
              class="flex-1 btn btn-secondary"
            >
              Cancel
            </button>
            <button
              type="submit"
              :disabled="!canSubmit || submitting"
              class="flex-1 btn btn-primary"
            >
              <template v-if="submitting">
                <ArrowPathIcon class="w-4 h-4 animate-spin mr-2" />
                Creating Proposal...
              </template>
              <template v-else>
                Create Proposal
              </template>
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { XMarkIcon, ArrowPathIcon, ExclamationTriangleIcon, InformationCircleIcon } from '@heroicons/vue/24/outline'
import { useNetworkStore } from '@/stores/network'
import { useWalletStore } from '@/stores/wallet'
import { useGovernanceStore } from '@/stores/governance'

const emit = defineEmits<{
  close: []
  success: []
}>()

const networkStore = useNetworkStore()
const walletStore = useWalletStore()
const governanceStore = useGovernanceStore()

const proposalType = ref('text')
const title = ref('')
const description = ref('')
const deposit = ref('10')
const recipientAddress = ref('')
const spendAmount = ref('')
const paramSubspace = ref('')
const paramKey = ref('')
const paramValue = ref('')
const upgradeName = ref('')
const upgradeHeight = ref('')
const upgradeInfo = ref('')
const cancelUpgradeName = ref('')
const submitting = ref(false)

const canSubmit = computed(() => {
  const baseValid = title.value.trim() && description.value.trim() && deposit.value && parseFloat(deposit.value) >= 10
  
  if (proposalType.value === 'community-spend') {
    return baseValid && recipientAddress.value.trim() && spendAmount.value && parseFloat(spendAmount.value) > 0
  }
  
  if (proposalType.value === 'param-change') {
    return baseValid && paramSubspace.value.trim() && paramKey.value.trim() && paramValue.value.trim()
  }
  
  if (proposalType.value === 'software-upgrade') {
    return baseValid && upgradeName.value.trim() && upgradeHeight.value && parseInt(upgradeHeight.value) > 0
  }
  
  if (proposalType.value === 'cancel-upgrade') {
    return baseValid
  }
  
  return baseValid
})

// Helper function to multiply by power of 10 without scientific notation
function multiplyByPowerOfTen(value: string | number, decimals: number): string {
  // Ensure we have a string
  const valueStr = String(value)
  
  // Convert to string manually to avoid scientific notation
  const [intPart, decPart = ''] = valueStr.split('.')
  const totalDecimalPlaces = decPart.length
  
  // If we need to add more zeros than we have decimal places, just append zeros
  if (decimals >= totalDecimalPlaces) {
    const zerosToAdd = decimals - totalDecimalPlaces
    return intPart + decPart + '0'.repeat(zerosToAdd)
  } else {
    // If we have more decimal places than needed, truncate
    const truncatedDecPart = decPart.slice(0, decimals)
    return intPart + truncatedDecPart
  }
}

async function submitProposal() {
  if (!canSubmit.value) return
  
  submitting.value = true
  try {
    // Convert deposit amount to base units (avoid scientific notation)
    console.log('Converting deposit:', deposit.value, 'with decimals:', networkStore.currentNetwork.decimals)
    const depositAmount = multiplyByPowerOfTen(deposit.value, networkStore.currentNetwork.decimals)
    console.log('Deposit result:', depositAmount)
    
    // Convert spend amount to base units if it's a community spend proposal
    const spendAmountBase = spendAmount.value ? 
      multiplyByPowerOfTen(spendAmount.value, networkStore.currentNetwork.decimals) :
      undefined
    
    // Submit proposal using governance store
    await governanceStore.submitProposal(
      proposalType.value as 'text' | 'community-spend' | 'param-change' | 'software-upgrade' | 'cancel-upgrade',
      title.value.trim(),
      description.value.trim(),
      depositAmount,
      recipientAddress.value.trim() || undefined,
      spendAmountBase,
      paramSubspace.value.trim() || undefined,
      paramKey.value.trim() || undefined,
      paramValue.value.trim() || undefined,
      upgradeName.value.trim() || undefined,
      upgradeHeight.value ? parseInt(upgradeHeight.value) : undefined,
      upgradeInfo.value.trim() || undefined,
      cancelUpgradeName.value.trim() || undefined
    )
    
    alert('Proposal submitted successfully!')
    emit('success')
    
  } catch (error) {
    console.error('Failed to submit proposal:', error)
    alert(`Failed to submit proposal: ${error instanceof Error ? error.message : 'Unknown error'}`)
  } finally {
    submitting.value = false
  }
}
</script>