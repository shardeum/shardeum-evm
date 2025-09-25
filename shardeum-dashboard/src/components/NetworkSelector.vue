<template>
  <div class="relative">
    <button
      @click="isOpen = !isOpen"
      class="flex items-center space-x-2 px-3 py-2 bg-white border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-shardeum-primary focus:border-transparent"
    >
      <div class="w-2 h-2 rounded-full bg-green-500"></div>
      <span>{{ networkStore.currentNetwork.name }}</span>
      <ChevronDownIcon class="w-4 h-4" />
    </button>

    <div
      v-if="isOpen"
      class="absolute right-0 mt-2 w-56 bg-white border border-gray-200 rounded-md shadow-lg z-50"
    >
      <div class="py-1">
        <button
          v-for="network in networkStore.availableNetworks"
          :key="network.id"
          @click="selectNetwork(network.id)"
          class="flex items-center w-full px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
          :class="{ 'bg-gray-50': network.id === networkStore.currentNetworkId }"
        >
          <div 
            class="w-2 h-2 rounded-full mr-3"
            :class="network.id === networkStore.currentNetworkId ? 'bg-green-500' : 'bg-gray-300'"
          ></div>
          <div class="flex-1">
            <div class="font-medium">{{ network.name }}</div>
            <div class="text-xs text-gray-500">{{ network.chainId }}</div>
          </div>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ChevronDownIcon } from '@heroicons/vue/20/solid'
import { useNetworkStore } from '@/stores/network'

const networkStore = useNetworkStore()
const isOpen = ref(false)

function selectNetwork(networkId: string) {
  networkStore.setNetwork(networkId)
  isOpen.value = false
}

function handleClickOutside(event: MouseEvent) {
  const target = event.target as Element
  if (!target.closest('.relative')) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>