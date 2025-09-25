<template>
  <div class="fixed inset-0 z-50 overflow-y-auto">
    <div class="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0">
      <div class="fixed inset-0 transition-opacity bg-gray-500 bg-opacity-75" @click="$emit('close')"></div>

      <div class="inline-block w-full max-w-md p-6 my-8 overflow-hidden text-left align-middle transition-all transform bg-white shadow-xl rounded-lg">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-medium text-gray-900">Delegate Tokens</h3>
          <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-6 h-6" />
          </button>
        </div>

        <div class="space-y-4">
          <!-- Validator Info -->
          <div class="p-4 bg-gray-50 rounded-lg">
            <div class="text-sm text-gray-600 mb-1">Delegating to:</div>
            <div class="font-medium text-gray-900">{{ validatorMoniker }}</div>
            <div class="text-sm text-gray-500 font-mono mt-1">{{ validatorAddress }}</div>
          </div>

          <!-- Amount Input -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">
              Amount to Delegate
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
            <div class="text-sm text-gray-500 mt-1">
              Available: {{ walletStore.formatBalance() }} {{ networkStore.currentNetwork.symbol }}
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
              @click="delegate"
              :disabled="!canDelegate || delegating"
              class="flex-1 btn btn-primary"
            >
              <template v-if="delegating">
                <ArrowPathIcon class="w-4 h-4 animate-spin mr-2" />
                Delegating...
              </template>
              <template v-else>
                Delegate
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
import { XMarkIcon, ArrowPathIcon } from '@heroicons/vue/24/outline'
import { useNetworkStore } from '@/stores/network'
import { useWalletStore } from '@/stores/wallet'

const props = defineProps<{
  validatorAddress: string
  validatorMoniker: string
}>()

const emit = defineEmits<{
  close: []
  success: []
}>()

const networkStore = useNetworkStore()
const walletStore = useWalletStore()

const amount = ref('')
const delegating = ref(false)

const amountError = computed(() => {
  if (!amount.value) return ''
  
  const amountNum = parseFloat(amount.value)
  if (isNaN(amountNum) || amountNum <= 0) {
    return 'Amount must be greater than 0'
  }
  
  const availableBalance = parseFloat(walletStore.balance) / Math.pow(10, networkStore.currentNetwork.decimals)
  const feeAmount = 0.005 // Estimated fee
  
  if (amountNum + feeAmount > availableBalance) {
    return 'Insufficient balance (including fees)'
  }
  
  return ''
})

const canDelegate = computed(() => {
  return amount.value && !amountError.value && props.validatorAddress && walletStore.isConnected
})

async function delegate() {
  if (!canDelegate.value) return
  
  delegating.value = true
  try {
    const amountToDelegate = Math.floor(parseFloat(amount.value) * Math.pow(10, networkStore.currentNetwork.decimals))
    
    console.log('Using manual delegation with ping.pub approach:', {
      validator: props.validatorAddress,
      amount: amountToDelegate.toString(),
      denom: networkStore.currentNetwork.baseDenom
    })
    
    const result = await walletStore.delegateTokensManually(
      props.validatorAddress,
      amountToDelegate.toString(),
      networkStore.currentNetwork.baseDenom
    )
    
    console.log('Manual delegation result:', result)
    
    if (result.code === 0) {
      emit('success')
    } else {
      throw new Error(result.rawLog || 'Transaction failed')
    }
  } catch (error) {
    console.error('Failed to delegate - Full error:', error)
    console.error('Error message:', error.message)
    console.error('Error stack:', error.stack)
    
    if (error.message.includes('Account not found')) {
      alert('Account not found on chain. Please send a small transaction first (like a transfer) to initialize your account, then try delegating again.')
    } else if (error.message.includes('CORS') || error.message.includes('Failed to fetch')) {
      alert(`Transaction failed due to CORS restrictions. 

To enable transactions, restart your network with CORS enabled:
1. Stop your current network: pkill -f 'shardeumd.*shardeum'
2. Restart with: make start-network
3. The updated scripts now enable RPC CORS automatically

If you're still getting CORS errors, the network may not have restarted with the new CORS settings.`)
    } else {
      alert(`Failed to delegate tokens: ${error.message}`)
    }
  } finally {
    delegating.value = false
  }
}

// Reset amount when validator changes
watch(() => props.validatorAddress, () => {
  amount.value = ''
})
</script>