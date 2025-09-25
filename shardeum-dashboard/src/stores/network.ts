import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { NetworkConfig } from '@/types/network'
import { NETWORK_CONFIGS } from '@/types/network'

export const useNetworkStore = defineStore('network', () => {
  const currentNetworkId = ref('local')
  
  const currentNetwork = computed((): NetworkConfig => {
    return NETWORK_CONFIGS[currentNetworkId.value] || NETWORK_CONFIGS.local
  })
  
  const availableNetworks = computed(() => {
    return Object.values(NETWORK_CONFIGS)
  })
  
  function setNetwork(networkId: string) {
    if (NETWORK_CONFIGS[networkId]) {
      currentNetworkId.value = networkId
      localStorage.setItem('shardeum-network', networkId)
    }
  }
  
  // Initialize from localStorage
  function initializeNetwork() {
    const savedNetwork = localStorage.getItem('shardeum-network')
    if (savedNetwork && NETWORK_CONFIGS[savedNetwork]) {
      currentNetworkId.value = savedNetwork
    }
  }
  
  return {
    currentNetworkId,
    currentNetwork,
    availableNetworks,
    setNetwork,
    initializeNetwork
  }
})