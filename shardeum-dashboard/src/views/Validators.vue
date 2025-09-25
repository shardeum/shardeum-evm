<template>
  <div class="space-y-6">
    <div class="flex justify-between items-center">
      <h1 class="text-2xl font-bold text-gray-900">Validators</h1>
      <div class="flex items-center space-x-4">
        <div class="text-sm text-gray-500">
          Total: {{ validators.length }} validators
        </div>
        <button
          @click="loadValidators"
          :disabled="loading"
          class="btn btn-secondary"
        >
          <ArrowPathIcon class="w-4 h-4 mr-2" :class="{ 'animate-spin': loading }" />
          Refresh
        </button>
      </div>
    </div>

    <!-- Validators Table -->
    <div class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Validator
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Voting Power
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Commission
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Status
              </th>
              <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr
              v-for="validator in validators"
              :key="validator.operatorAddress"
              class="hover:bg-gray-50"
            >
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="flex items-center">
                  <div class="flex-shrink-0 h-10 w-10">
                    <div class="h-10 w-10 rounded-full bg-shardeum-primary flex items-center justify-center text-white font-bold">
                      {{ (validator.description?.moniker || validator.operatorAddress.slice(-8)).charAt(0).toUpperCase() }}
                    </div>
                  </div>
                  <div class="ml-4">
                    <div class="text-sm font-medium text-gray-900">{{ validator.description?.moniker || validator.operatorAddress.slice(-8) }}</div>
                    <div class="text-sm text-gray-500 font-mono">
                      {{ validator.operatorAddress.slice(0, 20) }}...
                    </div>
                  </div>
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900">
                  {{ formatTokenAmount(validator.tokens) }} {{ networkStore.currentNetwork.symbol }}
                </div>
                <div class="text-sm text-gray-500">
                  {{ ((parseFloat(validator.tokens) / totalVotingPower) * 100).toFixed(2) }}%
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900">
                  {{ (parseFloat(validator.commission.commissionRates.rate) * 100).toFixed(2) }}%
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span
                  class="inline-flex px-2 py-1 text-xs font-semibold rounded-full"
                  :class="{
                    'bg-green-100 text-green-800': validator.status === 'BOND_STATUS_BONDED',
                    'bg-yellow-100 text-yellow-800': validator.status === 'BOND_STATUS_UNBONDING',
                    'bg-red-100 text-red-800': validator.status === 'BOND_STATUS_UNBONDED',
                    'bg-gray-100 text-gray-800': !['BOND_STATUS_BONDED', 'BOND_STATUS_UNBONDING', 'BOND_STATUS_UNBONDED'].includes(validator.status)
                  }"
                >
                  {{ getStatusText(validator.status) }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right">
                <router-link
                  :to="`/staking?validator=${validator.operatorAddress}`"
                  class="text-shardeum-primary hover:text-shardeum-primary/80 text-sm font-medium"
                  v-if="walletStore.isConnected && validator.status === 'BOND_STATUS_BONDED'"
                >
                  Delegate
                </router-link>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="loading" class="p-8 text-center">
        <ArrowPathIcon class="w-8 h-8 text-gray-400 animate-spin mx-auto mb-4" />
        <p class="text-gray-500">Loading validators...</p>
      </div>

      <div v-if="!loading && validators.length === 0" class="p-8 text-center">
        <UserGroupIcon class="w-12 h-12 text-gray-400 mx-auto mb-4" />
        <p class="text-gray-500">No validators found</p>
      </div>
    </div>

    <!-- Validator Statistics -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <div class="card p-6">
        <div class="flex items-center">
          <div class="p-3 rounded-md bg-green-100">
            <CheckCircleIcon class="h-6 w-6 text-green-600" />
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-500">Active Validators</p>
            <p class="text-2xl font-semibold text-gray-900">{{ activeValidators }}</p>
          </div>
        </div>
      </div>

      <div class="card p-6">
        <div class="flex items-center">
          <div class="p-3 rounded-md bg-blue-100">
            <CurrencyDollarIcon class="h-6 w-6 text-blue-600" />
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-500">Total Staked</p>
            <p class="text-2xl font-semibold text-gray-900">
              {{ formatTokenAmount(totalStaked.toString()) }}
            </p>
          </div>
        </div>
      </div>

      <div class="card p-6">
        <div class="flex items-center">
          <div class="p-3 rounded-md bg-purple-100">
            <ChartBarIcon class="h-6 w-6 text-purple-600" />
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-500">Avg Commission</p>
            <p class="text-2xl font-semibold text-gray-900">{{ avgCommission.toFixed(2) }}%</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  ArrowPathIcon,
  UserGroupIcon,
  CheckCircleIcon,
  CurrencyDollarIcon,
  ChartBarIcon
} from '@heroicons/vue/24/outline'
import { useNetworkStore } from '@/stores/network'
import { useWalletStore } from '@/stores/wallet'
import { useStakingStore } from '@/stores/staking'

interface Validator {
  operatorAddress: string
  consensusPubkey: any
  jailed: boolean
  status: string
  tokens: string
  delegatorShares: string
  description: {
    moniker: string
    identity: string
    website: string
    securityContact: string
    details: string
  }
  unbondingHeight: string
  unbondingTime: string
  commission: {
    commissionRates: {
      rate: string
      maxRate: string
      maxChangeRate: string
    }
    updateTime: string
  }
  minSelfDelegation: string
  moniker: string
}

const networkStore = useNetworkStore()
const walletStore = useWalletStore()
const stakingStore = useStakingStore()

const validators = computed(() => stakingStore.validators)
const loading = computed(() => stakingStore.isLoading)

const totalVotingPower = computed(() => {
  return validators.value.reduce((sum, v) => sum + parseFloat(v.tokens), 0)
})

const activeValidators = computed(() => {
  return validators.value.filter(v => v.status === 'BOND_STATUS_BONDED').length
})

const totalStaked = computed(() => {
  return validators.value.reduce((sum, v) => {
    if (v.status === 'BOND_STATUS_BONDED') {
      return sum + parseFloat(v.tokens)
    }
    return sum
  }, 0)
})

const avgCommission = computed(() => {
  if (validators.value.length === 0) return 0
  const total = validators.value.reduce((sum, v) => {
    return sum + parseFloat(v.commission.commissionRates.rate)
  }, 0)
  return (total / validators.value.length) * 100
})

onMounted(() => {
  loadValidators()
})

async function loadValidators() {
  await stakingStore.loadValidators()
}

function formatTokenAmount(amount: string): string {
  const num = parseFloat(amount) / Math.pow(10, networkStore.currentNetwork.decimals)
  return num.toLocaleString('en-US', { 
    minimumFractionDigits: 0, 
    maximumFractionDigits: 2 
  })
}

function getStatusText(status: string): string {
  switch (status) {
    case 'BOND_STATUS_BONDED':
      return 'Active'
    case 'BOND_STATUS_UNBONDING':
      return 'Unbonding'
    case 'BOND_STATUS_UNBONDED':
      return 'Inactive'
    default:
      return 'Unknown'
  }
}
</script>