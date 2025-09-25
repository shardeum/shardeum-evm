import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useWalletStore } from './wallet'
import { apiService, type Validator } from '@/services/api'

export const useStakingStore = defineStore('staking', () => {
  const walletStore = useWalletStore()
  
  const validators = ref<Validator[]>([])
  const delegations = ref<any[]>([])
  const unbondingDelegations = ref<any[]>([])
  const rewards = ref<any>({ rewards: [], total: [] })
  const isLoading = ref(false)
  
  const activeValidators = computed(() => {
    return validators.value.filter(v => v.status === 'BOND_STATUS_BONDED' && !v.jailed)
  })
  
  const totalDelegated = computed(() => {
    return delegations.value.reduce((total, delegation) => {
      return total + parseFloat(delegation.balance?.amount || '0')
    }, 0)
  })
  
  const totalRewards = computed(() => {
    const total = rewards.value.total || []
    return total.reduce((sum: number, reward: any) => {
      return sum + parseFloat(reward.amount || '0')
    }, 0)
  })
  
  async function loadValidators() {
    try {
      isLoading.value = true
      validators.value = await apiService.getValidators()
    } catch (error) {
      console.error('Failed to load validators:', error)
    } finally {
      isLoading.value = false
    }
  }
  
  async function loadDelegations() {
    if (!walletStore.address) return
    
    try {
      delegations.value = await apiService.getDelegations(walletStore.address)
    } catch (error) {
      console.error('Failed to load delegations:', error)
    }
  }
  
  async function loadUnbondingDelegations() {
    if (!walletStore.address) return
    
    try {
      unbondingDelegations.value = await apiService.getUnbondingDelegations(walletStore.address)
    } catch (error) {
      console.error('Failed to load unbonding delegations:', error)
    }
  }
  
  async function loadRewards() {
    if (!walletStore.address) return
    
    try {
      rewards.value = await apiService.getRewards(walletStore.address)
    } catch (error) {
      console.error('Failed to load rewards:', error)
    }
  }
  
  async function loadAll() {
    await Promise.all([
      loadValidators(),
      loadDelegations(),
      loadUnbondingDelegations(),
      loadRewards()
    ])
  }
  
  // Transaction functions would require the signing client
  async function delegate(validatorAddress: string, amount: string) {
    const signingClient = await walletStore.getSigningClient()
    if (!signingClient || !walletStore.address) {
      throw new Error('Wallet not connected')
    }
    
    // This would implement the actual delegation transaction
    console.log('Delegate:', { validatorAddress, amount })
    // TODO: Implement delegation transaction
  }
  
  async function undelegate(validatorAddress: string, amount: string) {
    const signingClient = await walletStore.getSigningClient()
    if (!signingClient || !walletStore.address) {
      throw new Error('Wallet not connected')
    }
    
    // This would implement the actual undelegation transaction
    console.log('Undelegate:', { validatorAddress, amount })
    // TODO: Implement undelegation transaction
  }
  
  async function claimRewards(validatorAddress?: string) {
    if (!walletStore.address) {
      throw new Error('Wallet not connected')
    }
    
    return await walletStore.claimRewardsManually(validatorAddress)
  }
  
  return {
    validators,
    activeValidators,
    delegations,
    unbondingDelegations,
    rewards,
    totalDelegated,
    totalRewards,
    isLoading,
    loadValidators,
    loadDelegations,
    loadUnbondingDelegations,
    loadRewards,
    loadAll,
    delegate,
    undelegate,
    claimRewards
  }
})