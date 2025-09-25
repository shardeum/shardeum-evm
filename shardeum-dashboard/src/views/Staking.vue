<template>
  <div class="space-y-6">
    <div class="flex justify-between items-center">
      <h1 class="text-2xl font-bold text-gray-900">Staking</h1>
      <div class="flex items-center space-x-4" v-if="walletStore.isConnected">
        <div class="text-sm text-gray-500">
          Available: {{ walletStore.formatBalance() }} {{ networkStore.currentNetwork.symbol }}
        </div>
        <button
          @click="loadDelegations"
          :disabled="loading"
          class="btn btn-secondary"
        >
          <ArrowPathIcon class="w-4 h-4 mr-2" :class="{ 'animate-spin': loading }" />
          Refresh
        </button>
      </div>
    </div>

    <!-- Connect Wallet Prompt -->
    <div v-if="!walletStore.isConnected" class="card p-8 text-center">
      <WalletIcon class="w-16 h-16 text-gray-400 mx-auto mb-4" />
      <h3 class="text-lg font-medium text-gray-900 mb-2">Connect Your Wallet</h3>
      <p class="text-gray-500 mb-6">Connect your Keplr wallet to start staking and earn rewards</p>
      <button @click="walletStore.connectKeplr" class="btn btn-primary">
        Connect Keplr Wallet
      </button>
    </div>

    <template v-else>
      <!-- My Delegations -->
      <div class="card">
        <div class="px-6 py-4 border-b border-gray-200 flex justify-between items-center">
          <h3 class="text-lg font-medium text-gray-900">My Delegations</h3>
          <div class="text-sm text-gray-500" v-if="delegations.length > 0">
            Total Staked: {{ totalStaked }} {{ networkStore.currentNetwork.symbol }}
          </div>
        </div>

        <div v-if="loading" class="p-8 text-center">
          <ArrowPathIcon class="w-8 h-8 text-gray-400 animate-spin mx-auto mb-4" />
          <p class="text-gray-500">Loading delegations...</p>
        </div>

        <div v-else-if="delegations.length === 0" class="p-8 text-center">
          <CurrencyDollarIcon class="w-12 h-12 text-gray-400 mx-auto mb-4" />
          <h4 class="text-lg font-medium text-gray-900 mb-2">No Delegations</h4>
          <p class="text-gray-500 mb-4">You haven't staked any tokens yet</p>
          <button @click="showDelegateModal = true" class="btn btn-primary">
            Start Staking
          </button>
        </div>

        <div v-else class="divide-y divide-gray-200">
          <div
            v-for="delegation in delegations"
            :key="delegation.validatorAddress"
            class="px-6 py-4"
          >
            <div class="flex items-center justify-between">
              <div class="flex items-center">
                <div class="flex-shrink-0 h-10 w-10">
                  <div class="h-10 w-10 rounded-full bg-shardeum-primary flex items-center justify-center text-white font-bold">
                    {{ delegation.validatorMoniker.charAt(0).toUpperCase() }}
                  </div>
                </div>
                <div class="ml-4">
                  <div class="text-sm font-medium text-gray-900">{{ delegation.validatorMoniker }}</div>
                  <div class="text-sm text-gray-500 font-mono">
                    {{ delegation.validatorAddress.slice(0, 20) }}...
                  </div>
                </div>
              </div>
              <div class="text-right">
                <div class="text-sm font-medium text-gray-900">
                  {{ formatTokenAmount(delegation.amount) }} {{ networkStore.currentNetwork.symbol }}
                </div>
                <div class="flex space-x-2 mt-1">
                  <button
                    @click="openDelegateModal(delegation.validatorAddress, delegation.validatorMoniker)"
                    class="text-xs text-shardeum-primary hover:text-shardeum-primary/80"
                  >
                    Delegate More
                  </button>
                  <button
                    @click="openUndelegateModal(delegation)"
                    class="text-xs text-red-600 hover:text-red-500"
                  >
                    Undelegate
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Rewards -->
      <div class="card" v-if="rewards.length > 0">
        <div class="px-6 py-4 border-b border-gray-200 flex justify-between items-center">
          <h3 class="text-lg font-medium text-gray-900">Staking Rewards</h3>
          <button
            @click="claimAllRewards"
            :disabled="claimingRewards || totalRewardsAmount === '0'"
            class="btn btn-primary btn-sm"
          >
            <template v-if="claimingRewards">
              <ArrowPathIcon class="w-4 h-4 animate-spin mr-2" />
              Claiming...
            </template>
            <template v-else>
              Claim All ({{ formatTokenAmount(totalRewardsAmount) }} {{ networkStore.currentNetwork.symbol }})
            </template>
          </button>
        </div>
        
        <div class="divide-y divide-gray-200">
          <div
            v-for="reward in rewards"
            :key="reward.validatorAddress"
            class="px-6 py-4 flex items-center justify-between"
          >
            <div class="text-sm font-medium text-gray-900">{{ reward.validatorMoniker }}</div>
            <div class="text-sm text-gray-900">
              {{ formatTokenAmount(reward.amount) }} {{ networkStore.currentNetwork.symbol }}
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- Delegate Modal -->
    <DelegateModal
      v-if="showDelegateModal"
      :validator-address="selectedValidator?.address || ''"
      :validator-moniker="selectedValidator?.moniker || ''"
      @close="showDelegateModal = false"
      @success="handleDelegateSuccess"
    />

    <!-- Undelegate Modal -->
    <UndelegateModal
      v-if="showUndelegateModal"
      :delegation="selectedDelegation"
      @close="showUndelegateModal = false"
      @success="handleUndelegateSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  ArrowPathIcon,
  WalletIcon,
  CurrencyDollarIcon
} from '@heroicons/vue/24/outline'
import { useNetworkStore } from '@/stores/network'
import { useWalletStore } from '@/stores/wallet'
import { useStakingStore } from '@/stores/staking'
import DelegateModal from '@/components/DelegateModal.vue'
import UndelegateModal from '@/components/UndelegateModal.vue'

interface Delegation {
  validatorAddress: string
  validatorMoniker: string
  amount: string
}

interface Reward {
  validatorAddress: string
  validatorMoniker: string
  amount: string
}

const route = useRoute()
const networkStore = useNetworkStore()
const walletStore = useWalletStore()
const stakingStore = useStakingStore()

const delegations = computed(() => stakingStore.delegations.map(d => ({
  validatorAddress: d.delegation?.validator_address || '',
  validatorMoniker: getValidatorMoniker(d.delegation?.validator_address || ''),
  amount: d.balance?.amount || '0'
})))

const rewards = computed(() => stakingStore.rewards.rewards?.map((r: any) => ({
  validatorAddress: r.validator_address,
  validatorMoniker: getValidatorMoniker(r.validator_address),
  amount: r.reward?.find((coin: any) => coin.denom === networkStore.currentNetwork.baseDenom)?.amount || '0'
})).filter((r: any) => parseFloat(r.amount) > 0) || [])

const loading = computed(() => stakingStore.isLoading)
const claimingRewards = ref(false)

const showDelegateModal = ref(false)
const showUndelegateModal = ref(false)
const selectedValidator = ref<{ address: string; moniker: string } | null>(null)
const selectedDelegation = ref<Delegation | null>(null)

const totalStaked = computed(() => {
  const total = delegations.value.reduce((sum, d) => sum + parseFloat(d.amount), 0)
  return formatTokenAmount(total.toString())
})

const totalRewardsAmount = computed(() => {
  return rewards.value.reduce((sum, r) => sum + parseFloat(r.amount), 0).toString()
})

function getValidatorMoniker(validatorAddress: string): string {
  const validator = stakingStore.validators.find(v => v.operatorAddress === validatorAddress)
  return validator?.description?.moniker || validatorAddress.slice(-8)
}

watch(() => walletStore.isConnected, (connected) => {
  if (connected) {
    loadDelegations()
  }
}, { immediate: true })

onMounted(() => {
  // Check if we should open delegate modal for a specific validator
  if (route.query.validator && walletStore.isConnected) {
    // We would need to get the validator moniker from the validators list
    selectedValidator.value = {
      address: route.query.validator as string,
      moniker: 'Validator'
    }
    showDelegateModal.value = true
  }
})

async function loadDelegations() {
  if (!walletStore.isConnected || !walletStore.address) return
  
  await Promise.all([
    stakingStore.loadValidators(),
    stakingStore.loadDelegations(),
    stakingStore.loadRewards()
  ])
}

function openDelegateModal(validatorAddress?: string, validatorMoniker?: string) {
  selectedValidator.value = {
    address: validatorAddress || '',
    moniker: validatorMoniker || 'Validator'
  }
  showDelegateModal.value = true
}

function openUndelegateModal(delegation: Delegation) {
  selectedDelegation.value = delegation
  showUndelegateModal.value = true
}

async function claimAllRewards() {
  if (rewards.value.length === 0) return
  
  claimingRewards.value = true
  try {
    const validatorAddresses = rewards.value.map(r => r.validatorAddress)
    await stakingStore.claimRewards(validatorAddresses[0]) // For now, claim from first validator
    
    await loadDelegations()
    await walletStore.updateBalance()
  } catch (error) {
    console.error('Failed to claim rewards:', error)
    alert('Failed to claim rewards. Please try again.')
  } finally {
    claimingRewards.value = false
  }
}

function handleDelegateSuccess() {
  showDelegateModal.value = false
  selectedValidator.value = null
  loadDelegations()
  walletStore.updateBalance()
}

function handleUndelegateSuccess() {
  showUndelegateModal.value = false
  selectedDelegation.value = null
  loadDelegations()
}

function formatTokenAmount(amount: string): string {
  const num = parseFloat(amount) / Math.pow(10, networkStore.currentNetwork.decimals)
  return num.toLocaleString('en-US', { 
    minimumFractionDigits: 0, 
    maximumFractionDigits: 6 
  })
}
</script>