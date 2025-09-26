<template>
  <div class="space-y-6">
    <!-- Network Overview -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-6">
      <div class="card p-6">
        <div class="flex items-center">
          <div class="p-3 rounded-md bg-shardeum-primary/10">
            <CubeIcon class="h-6 w-6 text-shardeum-primary" />
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-500">Latest Block</p>
            <p class="text-2xl font-semibold text-gray-900">{{ networkInfo.latestBlock || '...' }}</p>
          </div>
        </div>
      </div>

      <div class="card p-6">
        <div class="flex items-center">
          <div class="p-3 rounded-md bg-green-100">
            <UserGroupIcon class="h-6 w-6 text-green-600" />
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-500">Active Validators</p>
            <p class="text-2xl font-semibold text-gray-900">{{ networkInfo.activeValidators || '...' }}</p>
          </div>
        </div>
      </div>

      <div class="card p-6">
        <div class="flex items-center">
          <div class="p-3 rounded-md bg-blue-100">
            <CurrencyDollarIcon class="h-6 w-6 text-blue-600" />
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-500">Total Supply</p>
            <p class="text-2xl font-semibold text-gray-900">{{ formatTokenAmount(networkInfo.totalSupply) }}</p>
          </div>
        </div>
      </div>

      <div class="card p-6">
        <div class="flex items-center">
          <div class="p-3 rounded-md bg-purple-100">
            <BoltIcon class="h-6 w-6 text-purple-600" />
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-500">Avg Block Time</p>
            <p class="text-2xl font-semibold text-gray-900">{{ networkInfo.avgBlockTime || '...' }}s</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Recent Blocks and Recent Proposals -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Recent Blocks -->
      <div class="card">
        <div class="px-6 py-4 border-b border-gray-200">
          <h3 class="text-lg font-medium text-gray-900">Recent Blocks</h3>
        </div>
        <div class="divide-y divide-gray-200">
          <div
            v-for="block in recentBlocks"
            :key="block.height"
            class="px-6 py-4 hover:bg-gray-50"
          >
            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm font-medium text-gray-900"># {{ block.height }}</p>
                <p class="text-sm text-gray-500">{{ block.time }}</p>
              </div>
              <div class="text-right">
                <p class="text-sm text-gray-900">{{ block.txCount }} txs</p>
                <p class="text-sm text-gray-500">{{ block.size }} bytes</p>
              </div>
            </div>
          </div>
          <div v-if="recentBlocks.length === 0" class="px-6 py-8 text-center">
            <CubeIcon class="mx-auto h-12 w-12 text-gray-400" />
            <p class="mt-2 text-sm text-gray-500">Loading blocks...</p>
          </div>
        </div>
      </div>

      <!-- Recent Proposals -->
      <div class="card">
        <div class="px-6 py-4 border-b border-gray-200">
          <div class="flex items-center justify-between">
            <h3 class="text-lg font-medium text-gray-900">Recent Proposals</h3>
            <router-link
              to="/governance"
              class="text-sm text-shardeum-primary hover:text-shardeum-primary/80"
            >
              View All
            </router-link>
          </div>
        </div>
        <div class="divide-y divide-gray-200">
          <div
            v-for="proposal in recentProposals"
            :key="proposal.proposalId"
            class="px-6 py-4 hover:bg-gray-50 cursor-pointer"
            @click="$router.push('/governance')"
          >
            <div class="flex items-start justify-between">
              <div class="flex-1 pr-4">
                <div class="flex items-center space-x-2 mb-1">
                  <h4 class="text-sm font-medium text-gray-900 truncate">
                    #{{ proposal.proposalId }} {{ proposal.content.title }}
                  </h4>
                  <span
                    class="inline-flex px-2 py-1 text-xs font-semibold rounded-full flex-shrink-0"
                    :class="{
                      'bg-yellow-100 text-yellow-800': proposal.status === 2,
                      'bg-green-100 text-green-800': proposal.status === 3,
                      'bg-red-100 text-red-800': proposal.status === 4,
                      'bg-gray-100 text-gray-800': proposal.status === 1
                    }"
                  >
                    {{ getProposalStatusText(proposal.status) }}
                  </span>
                </div>
                <p class="text-xs text-gray-600 line-clamp-2">{{ proposal.content.description }}</p>
              </div>
            </div>
          </div>
          <div v-if="recentProposals.length === 0" class="px-6 py-8 text-center">
            <DocumentTextIcon class="mx-auto h-12 w-12 text-gray-400" />
            <p class="mt-2 text-sm text-gray-500">No recent proposals</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Network Status -->
    <div class="grid grid-cols-1 lg:grid-cols-1 gap-6">
      <div class="card">
        <div class="px-6 py-4 border-b border-gray-200">
          <h3 class="text-lg font-medium text-gray-900">Network Status</h3>
        </div>
        <div class="p-6 space-y-4">
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-500">Chain ID</span>
            <span class="text-sm font-medium text-gray-900">{{ networkStore.currentNetwork.chainId }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-500">Network</span>
            <span class="text-sm font-medium text-gray-900">{{ networkStore.currentNetwork.name }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-500">RPC Endpoint</span>
            <span class="text-sm font-medium text-gray-900 font-mono">{{ networkStore.currentNetwork.rpcEndpoint }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-500">API Endpoint</span>
            <span class="text-sm font-medium text-gray-900 font-mono">{{ networkStore.currentNetwork.apiEndpoint }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-500">Status</span>
            <div class="flex items-center">
              <div class="w-2 h-2 bg-green-500 rounded-full mr-2"></div>
              <span class="text-sm font-medium text-green-600">Online</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Quick Actions -->
    <div class="card p-6" v-if="walletStore.isConnected">
      <h3 class="text-lg font-medium text-gray-900 mb-4">Quick Actions</h3>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <router-link
          to="/staking"
          class="p-4 border border-gray-200 rounded-lg hover:border-shardeum-primary hover:shadow-sm transition-all"
        >
          <CurrencyDollarIcon class="h-8 w-8 text-shardeum-primary mb-2" />
          <h4 class="font-medium text-gray-900">Stake Tokens</h4>
          <p class="text-sm text-gray-500">Earn rewards by staking</p>
        </router-link>
        
        <router-link
          to="/governance"
          class="p-4 border border-gray-200 rounded-lg hover:border-shardeum-primary hover:shadow-sm transition-all"
        >
          <DocumentTextIcon class="h-8 w-8 text-shardeum-primary mb-2" />
          <h4 class="font-medium text-gray-900">Governance</h4>
          <p class="text-sm text-gray-500">Vote on proposals</p>
        </router-link>
        
        <router-link
          to="/validators"
          class="p-4 border border-gray-200 rounded-lg hover:border-shardeum-primary hover:shadow-sm transition-all"
        >
          <UserGroupIcon class="h-8 w-8 text-shardeum-primary mb-2" />
          <h4 class="font-medium text-gray-900">Validators</h4>
          <p class="text-sm text-gray-500">Explore validators</p>
        </router-link>
        
        <router-link
          to="/blocks"
          class="p-4 border border-gray-200 rounded-lg hover:border-shardeum-primary hover:shadow-sm transition-all"
        >
          <CubeIcon class="h-8 w-8 text-shardeum-primary mb-2" />
          <h4 class="font-medium text-gray-900">Blocks</h4>
          <p class="text-sm text-gray-500">Browse blockchain</p>
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import {
  CubeIcon,
  UserGroupIcon,
  CurrencyDollarIcon,
  BoltIcon,
  DocumentTextIcon
} from '@heroicons/vue/24/outline'
import { useNetworkStore } from '@/stores/network'
import { useWalletStore } from '@/stores/wallet'
import { useGovernanceStore } from '@/stores/governance'
import { apiService } from '@/services/api'
import { getProposalStatusText } from '@/types/governance'

const networkStore = useNetworkStore()
const walletStore = useWalletStore()
const governanceStore = useGovernanceStore()

const networkInfo = reactive({
  latestBlock: null as number | null,
  activeValidators: null as number | null,
  totalSupply: '0',
  avgBlockTime: null as number | null
})

const recentBlocks = ref<Array<{
  height: number
  time: string
  txCount: number
  size: number
}>>([])

const recentProposals = ref<Array<{
  proposalId: string
  content: {
    title: string
    description: string
  }
  status: number
}>>([])

let updateInterval: NodeJS.Timeout

onMounted(async () => {
  await loadNetworkInfo()
  await loadRecentBlocks()
  await loadRecentProposals()
  
  // Update every 30 seconds
  updateInterval = setInterval(() => {
    loadNetworkInfo()
    loadRecentBlocks()
    loadRecentProposals()
  }, 30000)
})

onUnmounted(() => {
  if (updateInterval) {
    clearInterval(updateInterval)
  }
})

async function loadNetworkInfo() {
  try {
    // Get node info from API
    const nodeInfo = await apiService.getNodeInfo()
    networkInfo.latestBlock = parseInt(nodeInfo.syncInfo.latestBlockHeight)
    
    // Get validator count
    try {
      const validators = await apiService.getValidators()
      networkInfo.activeValidators = validators.length
    } catch (e) {
      console.warn('Could not fetch validators:', e)
    }

    // Get total supply
    try {
      const totalSupply = await apiService.getTotalSupply()
      networkInfo.totalSupply = totalSupply
    } catch (e) {
      console.warn('Could not fetch total supply:', e)
    }

    // Get average block time
    try {
      const avgBlockTime = await apiService.getAverageBlockTime(10)
      networkInfo.avgBlockTime = avgBlockTime
    } catch (e) {
      console.warn('Could not calculate average block time:', e)
    }
  } catch (error) {
    console.error('Failed to load network info:', error)
  }
}

async function loadRecentBlocks() {
  try {
    const blocks = await apiService.getLatestBlocks(5)
    
    const formattedBlocks = blocks.map(block => ({
      height: parseInt(block?.header?.height || '0'),
      time: new Date(block?.header?.time || Date.now()).toLocaleTimeString(),
      txCount: block?.data?.txs?.length || 0,
      size: JSON.stringify(block).length // Rough size estimate
    }))
    
    recentBlocks.value = formattedBlocks
  } catch (error) {
    console.error('Failed to load recent blocks:', error)
  }
}

async function loadRecentProposals() {
  try {
    const allProposals = await apiService.getProposals()
    // Get the 3 most recent proposals
    recentProposals.value = allProposals
      .sort((a, b) => parseInt(b.proposalId) - parseInt(a.proposalId))
      .slice(0, 3)
  } catch (error) {
    console.error('Failed to load recent proposals:', error)
    recentProposals.value = []
  }
}

function formatTokenAmount(amount: string): string {
  const num = parseFloat(amount) / Math.pow(10, networkStore.currentNetwork.decimals)
  return num.toLocaleString('en-US', { 
    minimumFractionDigits: 0, 
    maximumFractionDigits: 0 
  })
}
</script>