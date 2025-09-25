<template>
  <div class="fixed inset-0 z-50 overflow-y-auto">
    <div class="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0">
      <div class="fixed inset-0 transition-opacity bg-gray-500 bg-opacity-75" @click="$emit('close')"></div>

      <div class="inline-block w-full max-w-4xl p-6 my-8 overflow-hidden text-left align-middle transition-all transform bg-white shadow-xl rounded-lg">
        <div class="flex items-center justify-between mb-6">
          <h3 class="text-xl font-medium text-gray-900">
            Block #{{ block.height }}
          </h3>
          <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-6 h-6" />
          </button>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <!-- Block Information -->
          <div class="card p-6">
            <h4 class="text-lg font-medium text-gray-900 mb-4">Block Information</h4>
            <div class="space-y-3">
              <div class="flex justify-between">
                <span class="text-sm text-gray-500">Height:</span>
                <span class="text-sm font-medium text-gray-900">{{ block.height }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-sm text-gray-500">Hash:</span>
                <span class="text-sm font-mono text-gray-900 break-all">{{ block.hash }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-sm text-gray-500">Time:</span>
                <span class="text-sm text-gray-900">{{ formatDateTime(block.time) }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-sm text-gray-500">Time Ago:</span>
                <span class="text-sm text-gray-900">{{ formatTimeAgo(block.time) }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-sm text-gray-500">Proposer:</span>
                <span class="text-sm font-mono text-gray-900">{{ block.proposer }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-sm text-gray-500">Transactions:</span>
                <span class="text-sm text-gray-900">{{ block.txCount }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-sm text-gray-500">Size:</span>
                <span class="text-sm text-gray-900">{{ formatSize(block.size) }}</span>
              </div>
              <div v-if="block.gasUsed" class="flex justify-between">
                <span class="text-sm text-gray-500">Gas Used:</span>
                <span class="text-sm text-gray-900">{{ formatNumber(block.gasUsed) }}</span>
              </div>
              <div v-if="block.gasWanted" class="flex justify-between">
                <span class="text-sm text-gray-500">Gas Wanted:</span>
                <span class="text-sm text-gray-900">{{ formatNumber(block.gasWanted) }}</span>
              </div>
            </div>
          </div>

          <!-- Navigation -->
          <div class="card p-6">
            <h4 class="text-lg font-medium text-gray-900 mb-4">Navigation</h4>
            <div class="space-y-3">
              <button
                @click="navigateToBlock(block.height - 1)"
                :disabled="block.height <= 1"
                class="w-full btn btn-secondary flex items-center justify-center"
              >
                <ChevronLeftIcon class="w-4 h-4 mr-2" />
                Previous Block (#{{ block.height - 1 }})
              </button>
              <button
                @click="navigateToBlock(block.height + 1)"
                class="w-full btn btn-secondary flex items-center justify-center"
              >
                Next Block (#{{ block.height + 1 }})
                <ChevronRightIcon class="w-4 h-4 ml-2" />
              </button>
            </div>

            <!-- Block Stats -->
            <div class="mt-6 pt-6 border-t border-gray-200">
              <h5 class="text-base font-medium text-gray-900 mb-3">Quick Stats</h5>
              <div class="grid grid-cols-2 gap-4">
                <div class="text-center p-3 bg-gray-50 rounded-lg">
                  <div class="text-2xl font-bold text-shardeum-primary">{{ block.txCount }}</div>
                  <div class="text-xs text-gray-600">Transactions</div>
                </div>
                <div class="text-center p-3 bg-gray-50 rounded-lg">
                  <div class="text-2xl font-bold text-shardeum-primary">{{ formatSize(block.size) }}</div>
                  <div class="text-xs text-gray-600">Block Size</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Transactions List -->
        <div class="mt-6">
          <div class="card">
            <div class="px-6 py-4 border-b border-gray-200">
              <h4 class="text-lg font-medium text-gray-900">
                Transactions ({{ block.txCount }})
              </h4>
            </div>
            
            <div v-if="loadingTransactions" class="p-8 text-center">
              <ArrowPathIcon class="w-8 h-8 text-gray-400 animate-spin mx-auto mb-4" />
              <p class="text-gray-500">Loading transactions...</p>
            </div>

            <div v-else-if="transactions.length === 0" class="p-8 text-center">
              <DocumentIcon class="w-12 h-12 text-gray-400 mx-auto mb-4" />
              <p class="text-gray-500">No transactions in this block</p>
            </div>

            <div v-else class="divide-y divide-gray-200">
              <div
                v-for="(tx, index) in transactions"
                :key="index"
                class="px-6 py-4 hover:bg-gray-50"
              >
                <div class="flex items-center justify-between">
                  <div class="flex-1">
                    <div class="text-sm font-mono text-gray-900 mb-1">
                      {{ tx.hash || `Transaction ${index + 1}` }}
                    </div>
                    <div class="text-sm text-gray-500">
                      {{ tx.type || 'Unknown Type' }}
                    </div>
                  </div>
                  <div class="text-right">
                    <div class="text-sm font-medium text-gray-900">
                      {{ tx.fee || '0' }} {{ networkStore.currentNetwork.symbol }}
                    </div>
                    <div class="text-sm text-gray-500">
                      {{ tx.success ? 'Success' : 'Failed' }}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  XMarkIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ArrowPathIcon,
  DocumentIcon
} from '@heroicons/vue/24/outline'
import { useNetworkStore } from '@/stores/network'

interface Block {
  height: number
  hash: string
  time: string
  proposer: string
  txCount: number
  size: number
  gasUsed?: string
  gasWanted?: string
}

interface Transaction {
  hash?: string
  type?: string
  fee?: string
  success?: boolean
}

const props = defineProps<{
  block: Block
}>()

const emit = defineEmits<{
  close: []
  navigateToBlock: [height: number]
}>()

const networkStore = useNetworkStore()

const transactions = ref<Transaction[]>([])
const loadingTransactions = ref(false)

onMounted(() => {
  loadTransactions()
})

async function loadTransactions() {
  if (props.block.txCount === 0) return
  
  loadingTransactions.value = true
  try {
    // For now, create mock transactions based on count
    // In a full implementation, this would fetch actual transaction data
    transactions.value = Array.from({ length: props.block.txCount }, (_, index) => ({
      hash: `tx_${props.block.height}_${index}`,
      type: 'Cosmos Transaction',
      fee: '0.005',
      success: true
    }))
  } catch (error) {
    console.error('Failed to load transactions:', error)
  } finally {
    loadingTransactions.value = false
  }
}

function navigateToBlock(height: number) {
  emit('navigateToBlock', height)
}

function formatDateTime(timeString: string): string {
  return new Date(timeString).toLocaleString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

function formatTimeAgo(timeString: string): string {
  const diff = Date.now() - new Date(timeString).getTime()
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  
  if (days > 0) return `${days} day${days > 1 ? 's' : ''} ago`
  if (hours > 0) return `${hours} hour${hours > 1 ? 's' : ''} ago`
  if (minutes > 0) return `${minutes} minute${minutes > 1 ? 's' : ''} ago`
  return `${seconds} second${seconds > 1 ? 's' : ''} ago`
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatNumber(value: string): string {
  return parseInt(value).toLocaleString()
}
</script>