<template>
  <div class="flex items-center space-x-3">
    <div v-if="walletStore.isConnected" class="flex items-center space-x-3">
      <!-- Balance Display -->
      <div class="text-sm text-gray-600">
        <span class="font-medium">{{ walletStore.formatBalance() }}</span>
        <span class="ml-1">{{ networkStore.currentNetwork.symbol }}</span>
      </div>
      
      <!-- Account Button -->
      <div class="relative">
        <button
          @click="isAccountMenuOpen = !isAccountMenuOpen"
          class="flex items-center space-x-2 px-3 py-2 bg-shardeum-primary text-white rounded-md hover:bg-shardeum-primary/90 focus:outline-none focus:ring-2 focus:ring-shardeum-primary focus:ring-offset-2"
        >
          <div class="w-2 h-2 bg-green-400 rounded-full"></div>
          <span class="text-sm font-medium">{{ walletStore.shortAddress }}</span>
          <ChevronDownIcon class="w-4 h-4" />
        </button>

        <!-- Account Dropdown -->
        <div
          v-if="isAccountMenuOpen"
          class="absolute right-0 mt-2 w-64 bg-white border border-gray-200 rounded-md shadow-lg z-50"
        >
          <div class="p-4 border-b border-gray-200">
            <div class="text-sm text-gray-600 mb-1">Account</div>
            <div class="font-mono text-sm text-gray-900 break-all">{{ walletStore.address }}</div>
          </div>
          <div class="p-4 border-b border-gray-200">
            <div class="text-sm text-gray-600 mb-1">Balance</div>
            <div class="font-medium">
              {{ walletStore.formatBalance() }} {{ networkStore.currentNetwork.symbol }}
            </div>
          </div>
          
          <!-- Network Selector -->
          <div class="p-4 border-b border-gray-200">
            <div class="text-sm text-gray-600 mb-2">Network</div>
            <div class="space-y-1">
              <button
                v-for="network in networkStore.availableNetworks"
                :key="network.id"
                @click="selectNetwork(network.id)"
                class="flex items-center w-full px-3 py-2 text-sm rounded-md hover:bg-gray-100"
                :class="{ 'bg-gray-100': network.id === networkStore.currentNetworkId }"
              >
                <div 
                  class="w-2 h-2 rounded-full mr-3"
                  :class="network.id === networkStore.currentNetworkId ? 'bg-green-500' : 'bg-gray-300'"
                ></div>
                <div class="flex-1 text-left">
                  <div class="font-medium">{{ network.name }}</div>
                  <div class="text-xs text-gray-500">{{ network.chainId }}</div>
                </div>
              </button>
            </div>
          </div>
          
          <div class="p-2">
            <button
              @click="refreshBalance"
              :disabled="isRefreshing"
              class="w-full px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-md flex items-center justify-center"
            >
              <ArrowPathIcon
                class="w-4 h-4 mr-2"
                :class="{ 'animate-spin': isRefreshing }"
              />
              Refresh Balance
            </button>
            <button
              @click="disconnect"
              class="w-full px-3 py-2 text-sm text-red-600 hover:bg-red-50 rounded-md"
            >
              Disconnect
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Connect Button -->
    <button
      v-else
      @click="connect"
      :disabled="walletStore.isConnecting"
      class="btn btn-primary flex items-center space-x-2"
    >
      <template v-if="walletStore.isConnecting">
        <ArrowPathIcon class="w-4 h-4 animate-spin" />
        <span>Connecting...</span>
      </template>
      <template v-else>
        <WalletIcon class="w-4 h-4" />
        <span>Connect Keplr</span>
      </template>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ChevronDownIcon, ArrowPathIcon } from '@heroicons/vue/20/solid'
import { WalletIcon } from '@heroicons/vue/24/outline'
import { useWalletStore } from '@/stores/wallet'
import { useNetworkStore } from '@/stores/network'

const walletStore = useWalletStore()
const networkStore = useNetworkStore()

const isAccountMenuOpen = ref(false)
const isRefreshing = ref(false)

async function connect() {
  const success = await walletStore.connectKeplr()
  if (!success) {
    alert('Failed to connect to Keplr. Please make sure Keplr is installed and try again.')
  }
}

function disconnect() {
  walletStore.disconnect()
  isAccountMenuOpen.value = false
}

function selectNetwork(networkId: string) {
  networkStore.setNetwork(networkId)
}

async function refreshBalance() {
  isRefreshing.value = true
  try {
    await walletStore.updateBalance()
  } finally {
    isRefreshing.value = false
  }
}

function handleClickOutside(event: MouseEvent) {
  const target = event.target as Element
  if (!target.closest('.relative')) {
    isAccountMenuOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>