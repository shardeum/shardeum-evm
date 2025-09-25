<template>
  <div class="fixed inset-0 z-50 overflow-y-auto">
    <div class="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0">
      <div class="fixed inset-0 transition-opacity bg-gray-500 bg-opacity-75" @click="$emit('close')"></div>

      <div class="inline-block w-full max-w-md p-6 my-8 overflow-hidden text-left align-middle transition-all transform bg-white shadow-xl rounded-lg">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-medium text-gray-900">Undelegate Tokens</h3>
          <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-6 h-6" />
          </button>
        </div>

        <div class="space-y-4">
          <!-- Validator Info -->
          <div class="p-4 bg-gray-50 rounded-lg">
            <div class="text-sm text-gray-600 mb-1">Undelegating from:</div>
            <div class="font-medium text-gray-900">{{ delegation?.validatorMoniker }}</div>
            <div class="text-sm text-gray-500 font-mono mt-1">{{ delegation?.validatorAddress }}</div>
          </div>

          <!-- Current Delegation Info -->
          <div class="p-3 bg-blue-50 rounded-lg">
            <div class="text-sm text-blue-800">
              <div class="flex justify-between">
                <span>Currently Delegated:</span>
                <span>{{ formatCurrentAmount() }} {{ networkStore.currentNetwork.symbol }}</span>
              </div>
            </div>
          </div>

          <!-- Amount Input -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">
              Amount to Undelegate
            </label>
            <div class="relative">
              <input
                v-model="amount"
                type="number"
                step="0.000001"
                placeholder="0.0"
                class="input pr-20"
                :class="{ 'border-red-500': amountError }"
              />
              <div class="absolute inset-y-0 right-0 flex items-center pr-3">
                <span class="text-sm text-gray-500">{{ networkStore.currentNetwork.symbol }}</span>
              </div>
            </div>
            <div v-if="amountError" class="text-sm text-red-600 mt-1">{{ amountError }}</div>
            <button
              @click="setMaxAmount"
              class="text-sm text-shardeum-primary hover:text-shardeum-primary/80 mt-1"
            >
              Use Max Amount
            </button>
          </div>

          <!-- Warning -->
          <div class="p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
            <div class="flex">
              <ExclamationTriangleIcon class="w-5 h-5 text-yellow-400 mr-2 mt-0.5" />
              <div class="text-sm text-yellow-800">
                <div class="font-medium mb-1">Unbonding Period: 21 days</div>
                <div>Your tokens will be locked for 21 days after undelegating and won't earn rewards during this period.</div>
              </div>
            </div>
          </div>

          <!-- Fee Estimate -->
          <div class="p-3 bg-blue-50 rounded-lg">
            <div class="text-sm text-blue-800">
              <div class="flex justify-between">
                <span>Transaction Fee:</span>
                <span>~0.005 {{ networkStore.currentNetwork.symbol }}</span>
              </div>
            </div>
          </div>

          <!-- Action Buttons -->
          <div class="flex space-x-3 pt-4">
            <button
              @click="$emit('close')"
              class="flex-1 btn btn-secondary"
            >
              Cancel
            </button>
            <button
              @click="undelegate"
              :disabled="!canUndelegate || undelegating"
              class="flex-1 btn bg-red-600 hover:bg-red-700 text-white"
            >
              <template v-if="undelegating">
                <ArrowPathIcon class="w-4 h-4 animate-spin mr-2" />
                Undelegating...
              </template>
              <template v-else>
                Undelegate
              </template>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { XMarkIcon, ArrowPathIcon, ExclamationTriangleIcon } from '@heroicons/vue/24/outline'
import { useNetworkStore } from '@/stores/network'
import { useWalletStore } from '@/stores/wallet'

interface Delegation {
  validatorAddress: string
  validatorMoniker: string
  amount: string
}

const props = defineProps<{
  delegation: Delegation | null
}>()

const emit = defineEmits<{
  close: []
  success: []
}>()

const networkStore = useNetworkStore()
const walletStore = useWalletStore()

const amount = ref('')
const undelegating = ref(false)

const amountError = computed(() => {
  if (!amount.value || !props.delegation) return ''
  
  const amountNum = parseFloat(amount.value)
  if (isNaN(amountNum) || amountNum <= 0) {
    return 'Amount must be greater than 0'
  }
  
  const currentDelegation = parseFloat(props.delegation.amount) / Math.pow(10, networkStore.currentNetwork.decimals)
  
  if (amountNum > currentDelegation) {
    return 'Amount exceeds current delegation'
  }
  
  return ''
})

const canUndelegate = computed(() => {
  return amount.value && !amountError.value && props.delegation && walletStore.isConnected
})

function formatCurrentAmount(): string {
  if (!props.delegation) return '0'
  const num = parseFloat(props.delegation.amount) / Math.pow(10, networkStore.currentNetwork.decimals)
  return num.toLocaleString('en-US', { 
    minimumFractionDigits: 0, 
    maximumFractionDigits: 6 
  })
}

function setMaxAmount() {
  if (!props.delegation) return
  const maxAmount = parseFloat(props.delegation.amount) / Math.pow(10, networkStore.currentNetwork.decimals)
  amount.value = maxAmount.toString()
}

async function undelegate() {
  if (!canUndelegate.value || !props.delegation) return
  
  undelegating.value = true
  try {
    const amountToUndelegate = Math.floor(parseFloat(amount.value) * Math.pow(10, networkStore.currentNetwork.decimals))
    
    const result = await walletStore.undelegateTokensManually(
      props.delegation.validatorAddress,
      amountToUndelegate.toString(),
      networkStore.currentNetwork.baseDenom
    )
    
    console.log('Undelegation successful:', result)
    emit('success')
  } catch (error) {
    console.error('Failed to undelegate:', error)
    alert('Failed to undelegate tokens. Please try again.')
  } finally {
    undelegating.value = false
  }
}

// Reset amount when delegation changes
watch(() => props.delegation, () => {
  amount.value = ''
})
</script>