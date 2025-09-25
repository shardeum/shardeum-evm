<template>
  <div class="space-y-6">
    <div class="flex justify-between items-center">
      <h1 class="text-2xl font-bold text-gray-900">Blocks</h1>
      <div class="flex items-center space-x-4">
        <div class="text-sm text-gray-500">
          Latest: {{ latestBlock || '...' }}
        </div>
        <button
          @click="loadBlocks"
          :disabled="loading"
          class="btn btn-secondary"
        >
          <ArrowPathIcon class="w-4 h-4 mr-2" :class="{ 'animate-spin': loading }" />
          Refresh
        </button>
      </div>
    </div>

    <!-- Search -->
    <div class="card p-4">
      <div class="flex space-x-4">
        <div class="flex-1">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search by block height or hash..."
            class="input"
            @keyup.enter="searchBlock"
          />
        </div>
        <button
          @click="searchBlock"
          :disabled="!searchQuery.trim() || searching"
          class="btn btn-primary"
        >
          <template v-if="searching">
            <ArrowPathIcon class="w-4 h-4 animate-spin mr-2" />
            Searching...
          </template>
          <template v-else>
            <MagnifyingGlassIcon class="w-4 h-4 mr-2" />
            Search
          </template>
        </button>
      </div>
    </div>

    <!-- Blocks List -->
    <div class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Height
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Hash
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Proposer
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Transactions
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Time
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Size
              </th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr
              v-for="block in blocks"
              :key="block.height"
              class="hover:bg-gray-50 cursor-pointer"
              @click="selectBlock(block)"
            >
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm font-medium text-shardeum-primary">
                  #{{ block.height }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm font-mono text-gray-900">
                  {{ block.hash.slice(0, 16) }}...
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900">
                  {{ block.proposer || 'Unknown' }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900">{{ block.txCount }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900">
                  {{ formatTime(block.time) }}
                </div>
                <div class="text-sm text-gray-500">
                  {{ formatTimeAgo(block.time) }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900">
                  {{ formatSize(block.size) }}
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="loading" class="p-8 text-center">
        <ArrowPathIcon class="w-8 h-8 text-gray-400 animate-spin mx-auto mb-4" />
        <p class="text-gray-500">Loading blocks...</p>
      </div>

      <div v-if="!loading && blocks.length === 0" class="p-8 text-center">
        <CubeIcon class="w-12 h-12 text-gray-400 mx-auto mb-4" />
        <p class="text-gray-500">No blocks found</p>
      </div>
    </div>

    <!-- Pagination -->
    <div class="flex justify-between items-center" v-if="blocks.length > 0">
      <div class="text-sm text-gray-500">
        Showing {{ blocks.length }} blocks
      </div>
      <div class="flex space-x-2">
        <button
          @click="loadOlderBlocks"
          :disabled="loading"
          class="btn btn-secondary"
        >
          Load Older Blocks
        </button>
      </div>
    </div>

    <!-- Block Detail Modal -->
    <BlockDetailModal
      v-if="selectedBlock"
      :block="selectedBlock"
      @close="selectedBlock = null"
      @navigateToBlock="navigateToBlock"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import {
  ArrowPathIcon,
  MagnifyingGlassIcon,
  CubeIcon
} from '@heroicons/vue/24/outline'
import { useNetworkStore } from '@/stores/network'
import { apiService } from '@/services/api'
import BlockDetailModal from '@/components/BlockDetailModal.vue'

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

const networkStore = useNetworkStore()

const blocks = ref<Block[]>([])
const latestBlock = ref<number | null>(null)
const loading = ref(false)
const searching = ref(false)
const searchQuery = ref('')
const selectedBlock = ref<Block | null>(null)

let updateInterval: NodeJS.Timeout

onMounted(async () => {
  await loadBlocks()
  
  // Update blocks every 30 seconds
  updateInterval = setInterval(loadBlocks, 30000)
})

onUnmounted(() => {
  if (updateInterval) {
    clearInterval(updateInterval)
  }
})

async function loadBlocks() {
  loading.value = true
  try {
    const blocksData = await apiService.getLatestBlocks(20)
    
    blocks.value = blocksData.map(block => ({
      height: parseInt(block?.header?.height || '0'),
      hash: block?.block_id?.hash || '',
      time: block?.header?.time || new Date().toISOString(),
      proposer: block?.header?.proposer_address || 'Unknown',
      txCount: block?.data?.txs?.length || 0,
      size: JSON.stringify(block).length, // Rough estimate
      gasUsed: block?.header?.gas_used,
      gasWanted: block?.header?.gas_wanted
    })).sort((a, b) => b.height - a.height)
    
    if (blocks.value.length > 0) {
      latestBlock.value = Math.max(...blocks.value.map(b => b.height))
    }
  } catch (error) {
    console.error('Failed to load blocks:', error)
  } finally {
    loading.value = false
  }
}

async function loadOlderBlocks() {
  // For now, we'll disable this functionality as it requires more complex API calls
  alert('Load older blocks functionality will be implemented soon.')
}

async function searchBlock() {
  if (!searchQuery.value.trim()) return
  
  searching.value = true
  try {
    const query = searchQuery.value.trim()
    
    // Try to search by height first (if it's a number)
    if (/^\d+$/.test(query)) {
      const height = parseInt(query)
      // Find the block in our current list
      const block = blocks.value.find(b => b.height === height)
      if (block) {
        selectedBlock.value = block
      } else {
        alert('Block not found in current list. Try refreshing blocks first.')
      }
    } else {
      alert('Hash search not implemented yet. Please search by block height.')
    }
  } catch (error) {
    console.error('Failed to search block:', error)
    alert('Failed to search block')
  } finally {
    searching.value = false
  }
}

function selectBlock(block: Block) {
  selectedBlock.value = block
}

async function navigateToBlock(height: number) {
  try {
    // First, try to find the block in our current blocks list
    let targetBlock = blocks.value.find(b => b.height === height)
    
    if (!targetBlock) {
      // Fetch the specific block from the API
      const blockData = await apiService.getBlockByHeight(height)
      
      if (blockData) {
        targetBlock = {
          height: parseInt(blockData?.header?.height || '0'),
          hash: blockData?.block_id?.hash || '',
          time: blockData?.header?.time || new Date().toISOString(),
          proposer: blockData?.header?.proposer_address || 'Unknown',
          txCount: blockData?.data?.txs?.length || 0,
          size: JSON.stringify(blockData).length,
          gasUsed: blockData?.header?.gas_used,
          gasWanted: blockData?.header?.gas_wanted
        }
      } else {
        throw new Error(`Block ${height} not found`)
      }
    }
    
    selectedBlock.value = targetBlock
  } catch (error) {
    console.error('Failed to navigate to block:', error)
    alert(`Failed to load block ${height}. It may not exist yet.`)
  }
}

function formatTime(timeString: string): string {
  return new Date(timeString).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function formatTimeAgo(timeString: string): string {
  const diff = Date.now() - new Date(timeString).getTime()
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  
  if (hours > 0) return `${hours}h ago`
  if (minutes > 0) return `${minutes}m ago`
  return `${seconds}s ago`
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
</script>