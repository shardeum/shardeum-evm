import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useNetworkStore } from './network'
import { SigningStargateClient, defaultRegistryTypes, StargateClient, AminoTypes, createDefaultAminoConverters } from '@cosmjs/stargate'
import { OfflineDirectSigner, OfflineAminoSigner, Registry, TxBodyEncodeObject, makeAuthInfoBytes, makeSignDoc } from '@cosmjs/proto-signing'
import { TxRaw } from 'cosmjs-types/cosmos/tx/v1beta1/tx'
import { MsgDelegate, MsgUndelegate } from 'cosmjs-types/cosmos/staking/v1beta1/tx'
import { MsgVote, MsgSubmitProposal, MsgDeposit } from 'cosmjs-types/cosmos/gov/v1beta1/tx'
import { VoteOption, TextProposal } from 'cosmjs-types/cosmos/gov/v1beta1/gov'
import { ParameterChangeProposal } from 'cosmjs-types/cosmos/params/v1beta1/params'
import { CommunityPoolSpendProposal } from 'cosmjs-types/cosmos/distribution/v1beta1/distribution'
import { SoftwareUpgradeProposal, CancelSoftwareUpgradeProposal, Plan } from 'cosmjs-types/cosmos/upgrade/v1beta1/upgrade'
import { Any } from 'cosmjs-types/google/protobuf/any'
import { PubKey } from 'cosmjs-types/cosmos/crypto/secp256k1/keys'
import { fromBech32, toHex, fromBase64 } from '@cosmjs/encoding'
import { encodeSecp256k1Pubkey, makeSignDoc as makeSignDocAmino } from '@cosmjs/amino'
import { SignMode } from 'cosmjs-types/cosmos/tx/signing/v1beta1/signing'
import { apiService } from '@/services/api'

export const useWalletStore = defineStore('wallet', () => {
  const networkStore = useNetworkStore()
  
  const isConnected = ref(false)
  const address = ref('')
  const balance = ref('0')
  const isConnecting = ref(false)
  
  const client = ref<SigningStargateClient | null>(null)

  // Helper function to get the correct API base URL
  function getApiBaseUrl(): string {
    // In development, use the Vite proxy (relative URLs)
    if (import.meta.env.DEV) {
      return ''
    }
    // In production, use the full endpoint URL
    return networkStore.currentNetwork.apiEndpoint
  }
  
  const shortAddress = computed(() => {
    if (!address.value) return ''
    return `${address.value.slice(0, 8)}...${address.value.slice(-6)}`
  })


  // Helper function to determine public key type based on chain ID (from ping.pub)
  function getKeyType(chainId: string) {
    switch (true) {
      case chainId.search(/\w+_\d+-\d+/g) > -1:   // ethermint like chain: evmos_9002-1
        return "/ethermint.crypto.v1.ethsecp256k1.PubKey"
      case chainId.startsWith("shardeum"):         // shardeum chains
        return "/ethermint.crypto.v1.ethsecp256k1.PubKey"
      case chainId.startsWith("injective"):
        return "/injective.crypto.v1beta1.ethsecp256k1.PubKey"
      case chainId.startsWith("stratos"):
        return "/stratos.crypto.v1.ethsecp256k1.PubKey"
      default:
        return "/cosmos.crypto.secp256k1.PubKey"
    }
  }
  
  async function connectKeplr() {
    if (!window.keplr) {
      alert('Please install Keplr extension')
      return false
    }
    
    try {
      isConnecting.value = true
      
      const { currentNetwork } = networkStore
      
      console.log('Suggesting chain to Keplr with config:', {
        chainId: currentNetwork.chainId,
        chainName: currentNetwork.name,
        symbol: currentNetwork.symbol,
        baseDenom: currentNetwork.baseDenom,
        decimals: currentNetwork.decimals
      })

      // Ensure we have valid denom values
      if (!currentNetwork.baseDenom || !currentNetwork.symbol) {
        throw new Error('Missing denom configuration for chain')
      }

      // Suggest the chain to Keplr
      await window.keplr.experimentalSuggestChain({
        chainId: currentNetwork.chainId,
        chainName: currentNetwork.name,
        rpc: currentNetwork.rpcEndpoint,
        rest: getApiBaseUrl() || currentNetwork.apiEndpoint,
        bip44: {
          coinType: 60, // Ethereum coin type for Ethermint compatibility
        },
        bech32Config: {
          bech32PrefixAccAddr: currentNetwork.bech32Prefix,
          bech32PrefixAccPub: `${currentNetwork.bech32Prefix}pub`,
          bech32PrefixValAddr: `${currentNetwork.bech32Prefix}valoper`,
          bech32PrefixValPub: `${currentNetwork.bech32Prefix}valoperpub`,
          bech32PrefixConsAddr: `${currentNetwork.bech32Prefix}valcons`,
          bech32PrefixConsPub: `${currentNetwork.bech32Prefix}valconspub`,
        },
        currencies: [
          {
            coinDenom: currentNetwork.symbol.toUpperCase(),
            coinMinimalDenom: currentNetwork.baseDenom.toLowerCase(),
            coinDecimals: currentNetwork.decimals,
          },
        ],
        feeCurrencies: [
          {
            coinDenom: currentNetwork.symbol.toUpperCase(),
            coinMinimalDenom: currentNetwork.baseDenom.toLowerCase(),
            coinDecimals: currentNetwork.decimals,
            gasPriceStep: {
              low: 0.000006,
              average: 0.000006,
              high: 0.000006,
            },
          },
        ],
        stakeCurrency: {
          coinDenom: currentNetwork.symbol.toUpperCase(),
          coinMinimalDenom: currentNetwork.baseDenom.toLowerCase(),
          coinDecimals: currentNetwork.decimals,
        },
        features: ['ibc-transfer', 'eth-address-gen', 'eth-key-sign'],
      })
      
      // Enable the chain
      await window.keplr.enable(currentNetwork.chainId)
      
      // Get the offline signer
      const offlineSigner: OfflineDirectSigner = window.getOfflineSigner(currentNetwork.chainId)
      const accounts = await offlineSigner.getAccounts()
      
      if (accounts.length === 0) {
        throw new Error('No accounts found')
      }
      
      
      address.value = accounts[0].address
      
      // Skip SigningStargateClient connection for now to avoid CORS issues
      // We'll create it only when needed for transactions
      client.value = null
      
      // Get balance using our API service
      await updateBalance()
      
      isConnected.value = true
      return true
      
    } catch (error) {
      console.error('Failed to connect to Keplr:', error)
      return false
    } finally {
      isConnecting.value = false
    }
  }
  
  async function updateBalance() {
    if (!address.value) return
    
    try {
      const { currentNetwork } = networkStore
      const balances = await apiService.getBalance(address.value, currentNetwork.baseDenom)
      
      if (balances.length > 0) {
        balance.value = balances[0].amount
      } else {
        balance.value = '0'
      }
    } catch (error) {
      console.error('Failed to get balance:', error)
      balance.value = '0'
    }
  }
  
  async function getSigningClient(): Promise<SigningStargateClient | null> {
    if (!isConnected.value || !window.keplr) {
      console.log('Not connected or no Keplr')
      return null
    }
    
    try {
      console.log('Creating signing client...')
      const { currentNetwork } = networkStore
      const offlineSigner: OfflineDirectSigner = window.getOfflineSigner(currentNetwork.chainId)
      
      // Create registry with all default Cosmos SDK message types
      const registry = new Registry(defaultRegistryTypes)
      
      // Create signing client without custom account parser (like ping.pub)
      const options = {
        registry
      }
      
      // Create signing client - CORS should now be enabled on RPC endpoints
      client.value = await SigningStargateClient.connectWithSigner(
        currentNetwork.rpcEndpoint,
        offlineSigner,
        options
      )
      
      console.log('Signing client created successfully')
      return client.value
    } catch (error) {
      console.error('Failed to create signing client:', error)
      return null
    }
  }

  // Manual delegation function following ping.pub's exact approach
  async function delegateTokensManually(
    validatorAddress: string, 
    amount: string, 
    denom: string
  ): Promise<any> {
    if (!isConnected.value || !window.keplr) {
      throw new Error('Wallet not connected')
    }

    try {
      const { currentNetwork } = networkStore
      const signerAddress = address.value

      // Get account info from chain using REST API (exactly like ping.pub)
      const accountResponse = await fetch(`${getApiBaseUrl()}/cosmos/auth/v1beta1/accounts/${signerAddress}`)
      if (!accountResponse.ok) {
        throw new Error('Account not found on chain')
      }
      
      const accountData = await accountResponse.json()
      
      // Use ping.pub's findField approach to extract account_number and sequence
      const findField = (obj: any, name: string): any => {
        if (!obj) return undefined
        const list = Object.keys(obj).filter(x => x && !x.startsWith("@"))
        if (list.includes(name)) {
          return obj[name]
        }
        for (let i = 0; i < list.length; i++) {
          const field = obj[list[i]]
          if (typeof field === 'string') continue
          if (Array.isArray(field)) continue
          const sub = findField(field, name)
          if (sub) return sub
        }
        return undefined
      }

      const accountNumber = Number(findField(accountData, "account_number"))
      const sequence = Number(findField(accountData, "sequence"))

      if (accountNumber === undefined || sequence === undefined) {
        throw new Error('Could not extract account number or sequence from account data')
      }

      console.log('Account info for manual delegation:', { accountNumber, sequence })

      // Create the transaction object exactly like ping.pub does
      const transaction = {
        chainId: currentNetwork.chainId,
        signerAddress: signerAddress,
        messages: [
          {
            typeUrl: '/cosmos.staking.v1beta1.MsgDelegate',
            value: {
              delegatorAddress: signerAddress,
              validatorAddress: validatorAddress,
              amount: {
                denom: denom,
                amount: amount
              }
            }
          }
        ],
        fee: {
          gas: "200000",
          amount: [
            { amount: "5000", denom: denom }
          ]
        },
        memo: "",
        signerData: {
          accountNumber: accountNumber,
          sequence: sequence,
          chainId: currentNetwork.chainId
        }
      }

      console.log('Created transaction object:', transaction)

      // Create a Keplr wallet instance like ping.pub does
      const offlineSigner: OfflineDirectSigner = window.getOfflineSigner(currentNetwork.chainId)

      // Use ping.pub's signing approach - construct the signDoc manually but let Keplr handle it
      const registry = new Registry(defaultRegistryTypes)

      // Wrap message in Any type for encoding
      const msgAny = Any.fromPartial({
        typeUrl: '/cosmos.staking.v1beta1.MsgDelegate',
        value: MsgDelegate.encode(MsgDelegate.fromPartial(transaction.messages[0].value)).finish()
      })

      // Create transaction body
      const txBodyEncodeObject: TxBodyEncodeObject = {
        typeUrl: "/cosmos.tx.v1beta1.TxBody",
        value: {
          messages: [msgAny],
          memo: transaction.memo,
        },
      }

      const txBodyBytes = registry.encode(txBodyEncodeObject)

      // Get signer accounts for pubkey
      const accounts = await offlineSigner.getAccounts()
      if (accounts.length === 0) {
        throw new Error('No accounts found')
      }

      // Create public key with correct type for this chain
      const pubkey = Any.fromPartial({
        typeUrl: getKeyType(currentNetwork.chainId),
        value: PubKey.encode({
          key: accounts[0].pubkey,
        }).finish()
      })

      const gasLimit = Number(transaction.fee.gas)
      const authInfoBytes = makeAuthInfoBytes(
        [{ pubkey, sequence: transaction.signerData.sequence }],
        transaction.fee.amount,
        gasLimit
      )

      const signDoc = makeSignDoc(
        txBodyBytes, 
        authInfoBytes, 
        transaction.chainId, 
        transaction.signerData.accountNumber
      )

      console.log('About to sign with Keplr:', {
        signerAddress: transaction.signerAddress,
        chainId: transaction.chainId,
        accountNumber: transaction.signerData.accountNumber,
        sequence: transaction.signerData.sequence,
        keyType: getKeyType(currentNetwork.chainId)
      })

      // Try amino signing first (more compatible with Ethermint chains)
      let signature: any, signed: any;
      
      try {
        // Try amino signing first
        const aminoTypes = new AminoTypes(createDefaultAminoConverters())
        const aminoSigner = window.getOfflineSignerOnlyAmino(currentNetwork.chainId) as OfflineAminoSigner
        
        const aminoMsgs = transaction.messages.map((msg) => aminoTypes.toAmino(msg))
        const signDocAmino = makeSignDocAmino(
          aminoMsgs, 
          transaction.fee, 
          transaction.chainId, 
          transaction.memo, 
          transaction.signerData.accountNumber, 
          transaction.signerData.sequence
        )
        
        console.log('Trying amino signing...')
        const aminoResult = await aminoSigner.signAmino(signerAddress, signDocAmino)
        
        // Convert amino result to direct signing format
        const signedTxBody = {
          messages: aminoResult.signed.msgs.map((msg) => aminoTypes.fromAmino(msg)),
          memo: aminoResult.signed.memo,
        }
        
        const signedTxBodyEncodeObject: TxBodyEncodeObject = {
          typeUrl: "/cosmos.tx.v1beta1.TxBody",
          value: signedTxBody,
        }
        
        const signedTxBodyBytes = registry.encode(signedTxBodyEncodeObject)
        const signedGasLimit = Number(aminoResult.signed.fee.gas)
        const signedSequence = Number(aminoResult.signed.sequence)
        
        const signedAuthInfoBytes = makeAuthInfoBytes(
          [{ pubkey, sequence: signedSequence }],
          aminoResult.signed.fee.amount,
          signedGasLimit,
          aminoResult.signed.fee.granter,
          aminoResult.signed.fee.payer,
          SignMode.SIGN_MODE_LEGACY_AMINO_JSON,
        )
        
        signed = {
          bodyBytes: signedTxBodyBytes,
          authInfoBytes: signedAuthInfoBytes,
        }
        signature = aminoResult.signature
        
        console.log('Amino signing successful')
        
      } catch (aminoError) {
        console.log('Amino signing failed, trying direct signing:', aminoError)
        
        // Fallback to direct signing
        const result = await offlineSigner.signDirect(signerAddress, signDoc)
        signature = result.signature
        signed = result.signed
        
        console.log('Direct signing successful')
      }

      // Create final transaction
      const txRaw = TxRaw.fromPartial({
        bodyBytes: signed.bodyBytes,
        authInfoBytes: signed.authInfoBytes,
        signatures: [fromBase64(signature.signature)],
      })

      // Broadcast using ping.pub's approach - via REST API
      const txBytes = TxRaw.encode(txRaw).finish()
      const txBase64 = btoa(String.fromCharCode(...txBytes))
      
      const broadcastRequest = {
        tx_bytes: txBase64,
        mode: 'BROADCAST_MODE_SYNC'
      }

      const broadcastResponse = await fetch(`${getApiBaseUrl()}/cosmos/tx/v1beta1/txs`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(broadcastRequest)
      })

      if (!broadcastResponse.ok) {
        throw new Error('Failed to broadcast transaction')
      }

      const result = await broadcastResponse.json()

      console.log('Manual delegation result:', result)
      
      // Check for broadcast errors
      if (result.code && result.code !== 0) {
        throw new Error(result.message || 'Broadcast error')
      }

      if (result.tx_response && result.tx_response.code !== 0) {
        throw new Error(result.tx_response.raw_log || 'Transaction failed')
      }

      return result.tx_response || result

    } catch (error) {
      console.error('Manual delegation failed:', error)
      throw error
    }
  }

  // Manual undelegation function following same ping.pub approach
  async function undelegateTokensManually(
    validatorAddress: string, 
    amount: string, 
    denom: string
  ): Promise<any> {
    if (!isConnected.value || !window.keplr) {
      throw new Error('Wallet not connected')
    }

    try {
      const { currentNetwork } = networkStore
      const signerAddress = address.value

      // Get account info from chain using REST API (exactly like ping.pub)
      const accountResponse = await fetch(`${getApiBaseUrl()}/cosmos/auth/v1beta1/accounts/${signerAddress}`)
      if (!accountResponse.ok) {
        throw new Error('Account not found on chain')
      }
      
      const accountData = await accountResponse.json()
      
      // Use ping.pub's findField approach to extract account_number and sequence
      const findField = (obj: any, name: string): any => {
        if (!obj) return undefined
        const list = Object.keys(obj).filter(x => x && !x.startsWith("@"))
        if (list.includes(name)) {
          return obj[name]
        }
        for (let i = 0; i < list.length; i++) {
          const field = obj[list[i]]
          if (typeof field === 'string') continue
          if (Array.isArray(field)) continue
          const sub = findField(field, name)
          if (sub) return sub
        }
        return undefined
      }

      const accountNumber = Number(findField(accountData, "account_number"))
      const sequence = Number(findField(accountData, "sequence"))

      if (accountNumber === undefined || sequence === undefined) {
        throw new Error('Could not extract account number or sequence from account data')
      }

      console.log('Account info for manual undelegation:', { accountNumber, sequence })

      // Create the transaction object exactly like ping.pub does
      const transaction = {
        chainId: currentNetwork.chainId,
        signerAddress: signerAddress,
        messages: [
          {
            typeUrl: '/cosmos.staking.v1beta1.MsgUndelegate',
            value: {
              delegatorAddress: signerAddress,
              validatorAddress: validatorAddress,
              amount: {
                denom: denom,
                amount: amount
              }
            }
          }
        ],
        fee: {
          gas: "200000",
          amount: [
            { amount: "5000", denom: denom }
          ]
        },
        memo: "",
        signerData: {
          accountNumber: accountNumber,
          sequence: sequence,
          chainId: currentNetwork.chainId
        }
      }

      console.log('Created undelegation transaction object:', transaction)

      // Create a Keplr wallet instance like ping.pub does
      const offlineSigner: OfflineDirectSigner = window.getOfflineSigner(currentNetwork.chainId)

      // Use ping.pub's signing approach - construct the signDoc manually but let Keplr handle it
      const registry = new Registry(defaultRegistryTypes)

      // Wrap message in Any type for encoding
      const msgAny = Any.fromPartial({
        typeUrl: '/cosmos.staking.v1beta1.MsgUndelegate',
        value: MsgUndelegate.encode(MsgUndelegate.fromPartial(transaction.messages[0].value)).finish()
      })

      // Create transaction body
      const txBodyEncodeObject: TxBodyEncodeObject = {
        typeUrl: "/cosmos.tx.v1beta1.TxBody",
        value: {
          messages: [msgAny],
          memo: transaction.memo,
        },
      }

      const txBodyBytes = registry.encode(txBodyEncodeObject)

      // Get signer accounts for pubkey
      const accounts = await offlineSigner.getAccounts()
      if (accounts.length === 0) {
        throw new Error('No accounts found')
      }

      // Create public key with correct type for this chain
      const pubkey = Any.fromPartial({
        typeUrl: getKeyType(currentNetwork.chainId),
        value: PubKey.encode({
          key: accounts[0].pubkey,
        }).finish()
      })

      const gasLimit = Number(transaction.fee.gas)
      const authInfoBytes = makeAuthInfoBytes(
        [{ pubkey, sequence: transaction.signerData.sequence }],
        transaction.fee.amount,
        gasLimit
      )

      const signDoc = makeSignDoc(
        txBodyBytes, 
        authInfoBytes, 
        transaction.chainId, 
        transaction.signerData.accountNumber
      )

      console.log('About to sign undelegation with Keplr:', {
        signerAddress: transaction.signerAddress,
        chainId: transaction.chainId,
        accountNumber: transaction.signerData.accountNumber,
        sequence: transaction.signerData.sequence,
        keyType: getKeyType(currentNetwork.chainId)
      })

      // Try amino signing first (more compatible with Ethermint chains)
      let signature: any, signed: any;
      
      try {
        // Try amino signing first
        const aminoTypes = new AminoTypes(createDefaultAminoConverters())
        const aminoSigner = window.getOfflineSignerOnlyAmino(currentNetwork.chainId) as OfflineAminoSigner
        
        const aminoMsgs = transaction.messages.map((msg) => aminoTypes.toAmino(msg))
        const signDocAmino = makeSignDocAmino(
          aminoMsgs, 
          transaction.fee, 
          transaction.chainId, 
          transaction.memo, 
          transaction.signerData.accountNumber, 
          transaction.signerData.sequence
        )
        
        console.log('Trying amino signing for undelegation...')
        const aminoResult = await aminoSigner.signAmino(signerAddress, signDocAmino)
        
        // Convert amino result to direct signing format
        const signedTxBody = {
          messages: aminoResult.signed.msgs.map((msg) => aminoTypes.fromAmino(msg)),
          memo: aminoResult.signed.memo,
        }
        
        const signedTxBodyEncodeObject: TxBodyEncodeObject = {
          typeUrl: "/cosmos.tx.v1beta1.TxBody",
          value: signedTxBody,
        }
        
        const signedTxBodyBytes = registry.encode(signedTxBodyEncodeObject)
        const signedGasLimit = Number(aminoResult.signed.fee.gas)
        const signedSequence = Number(aminoResult.signed.sequence)
        
        const signedAuthInfoBytes = makeAuthInfoBytes(
          [{ pubkey, sequence: signedSequence }],
          aminoResult.signed.fee.amount,
          signedGasLimit,
          aminoResult.signed.fee.granter,
          aminoResult.signed.fee.payer,
          SignMode.SIGN_MODE_LEGACY_AMINO_JSON,
        )
        
        signed = {
          bodyBytes: signedTxBodyBytes,
          authInfoBytes: signedAuthInfoBytes,
        }
        signature = aminoResult.signature
        
        console.log('Amino signing successful for undelegation')
        
      } catch (aminoError) {
        console.log('Amino signing failed, trying direct signing for undelegation:', aminoError)
        
        // Fallback to direct signing
        const result = await offlineSigner.signDirect(signerAddress, signDoc)
        signature = result.signature
        signed = result.signed
        
        console.log('Direct signing successful for undelegation')
      }

      // Create final transaction
      const txRaw = TxRaw.fromPartial({
        bodyBytes: signed.bodyBytes,
        authInfoBytes: signed.authInfoBytes,
        signatures: [fromBase64(signature.signature)],
      })

      // Broadcast using ping.pub's approach - via REST API
      const txBytes = TxRaw.encode(txRaw).finish()
      const txBase64 = btoa(String.fromCharCode(...txBytes))
      
      const broadcastRequest = {
        tx_bytes: txBase64,
        mode: 'BROADCAST_MODE_SYNC'
      }

      const broadcastResponse = await fetch(`${getApiBaseUrl()}/cosmos/tx/v1beta1/txs`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(broadcastRequest)
      })

      if (!broadcastResponse.ok) {
        throw new Error('Failed to broadcast undelegation transaction')
      }

      const result = await broadcastResponse.json()

      console.log('Manual undelegation result:', result)
      
      // Check for broadcast errors
      if (result.code && result.code !== 0) {
        throw new Error(result.message || 'Broadcast error')
      }

      if (result.tx_response && result.tx_response.code !== 0) {
        throw new Error(result.tx_response.raw_log || 'Undelegation transaction failed')
      }

      return result.tx_response || result

    } catch (error) {
      console.error('Manual undelegation failed:', error)
      throw error
    }
  }

  // Manual voting function following same ping.pub approach
  async function voteOnProposalManually(
    proposalId: string,
    option: number // VoteOption enum value
  ): Promise<any> {
    if (!isConnected.value || !window.keplr) {
      throw new Error('Wallet not connected')
    }

    try {
      const { currentNetwork } = networkStore
      const signerAddress = address.value

      // Get account info from chain using REST API (exactly like ping.pub)
      const accountResponse = await fetch(`${getApiBaseUrl()}/cosmos/auth/v1beta1/accounts/${signerAddress}`)
      if (!accountResponse.ok) {
        throw new Error('Account not found on chain')
      }
      
      const accountData = await accountResponse.json()
      
      // Use ping.pub's findField approach to extract account_number and sequence
      const findField = (obj: any, name: string): any => {
        if (!obj) return undefined
        const list = Object.keys(obj).filter(x => x && !x.startsWith("@"))
        if (list.includes(name)) {
          return obj[name]
        }
        for (let i = 0; i < list.length; i++) {
          const field = obj[list[i]]
          if (typeof field === 'string') continue
          if (Array.isArray(field)) continue
          const sub = findField(field, name)
          if (sub) return sub
        }
        return undefined
      }

      const accountNumber = Number(findField(accountData, "account_number"))
      const sequence = Number(findField(accountData, "sequence"))

      if (accountNumber === undefined || sequence === undefined) {
        throw new Error('Could not extract account number or sequence from account data')
      }

      console.log('Account info for manual vote:', { accountNumber, sequence })

      // Create the transaction object exactly like ping.pub does
      const transaction = {
        chainId: currentNetwork.chainId,
        signerAddress: signerAddress,
        messages: [
          {
            typeUrl: '/cosmos.gov.v1beta1.MsgVote',
            value: {
              proposalId: proposalId,
              voter: signerAddress,
              option: option
            }
          }
        ],
        fee: {
          gas: "200000",
          amount: [
            { amount: "5000", denom: currentNetwork.baseDenom }
          ]
        },
        memo: "",
        signerData: {
          accountNumber: accountNumber,
          sequence: sequence,
          chainId: currentNetwork.chainId
        }
      }

      console.log('Created vote transaction object:', transaction)

      // Create a Keplr wallet instance like ping.pub does
      const offlineSigner: OfflineDirectSigner = window.getOfflineSigner(currentNetwork.chainId)

      // Use ping.pub's signing approach - construct the signDoc manually but let Keplr handle it
      const registry = new Registry(defaultRegistryTypes)

      // Wrap message in Any type for encoding
      const msgAny = Any.fromPartial({
        typeUrl: '/cosmos.gov.v1beta1.MsgVote',
        value: MsgVote.encode(MsgVote.fromPartial(transaction.messages[0].value)).finish()
      })

      // Create transaction body
      const txBodyEncodeObject: TxBodyEncodeObject = {
        typeUrl: "/cosmos.tx.v1beta1.TxBody",
        value: {
          messages: [msgAny],
          memo: transaction.memo,
        },
      }

      const txBodyBytes = registry.encode(txBodyEncodeObject)

      // Get signer accounts for pubkey
      const accounts = await offlineSigner.getAccounts()
      if (accounts.length === 0) {
        throw new Error('No accounts found')
      }

      // Create public key with correct type for this chain
      const pubkey = Any.fromPartial({
        typeUrl: getKeyType(currentNetwork.chainId),
        value: PubKey.encode({
          key: accounts[0].pubkey,
        }).finish()
      })

      const gasLimit = Number(transaction.fee.gas)
      const authInfoBytes = makeAuthInfoBytes(
        [{ pubkey, sequence: transaction.signerData.sequence }],
        transaction.fee.amount,
        gasLimit
      )

      const signDoc = makeSignDoc(
        txBodyBytes, 
        authInfoBytes, 
        transaction.chainId, 
        transaction.signerData.accountNumber
      )

      console.log('About to sign vote with Keplr:', {
        signerAddress: transaction.signerAddress,
        chainId: transaction.chainId,
        proposalId: proposalId,
        option: option,
        accountNumber: transaction.signerData.accountNumber,
        sequence: transaction.signerData.sequence,
        keyType: getKeyType(currentNetwork.chainId)
      })

      // Try amino signing first (more compatible with Ethermint chains)
      let signature: any, signed: any;
      
      try {
        // Try amino signing first
        const aminoTypes = new AminoTypes(createDefaultAminoConverters())
        const aminoSigner = window.getOfflineSignerOnlyAmino(currentNetwork.chainId) as OfflineAminoSigner
        
        const aminoMsgs = transaction.messages.map((msg) => aminoTypes.toAmino(msg))
        const signDocAmino = makeSignDocAmino(
          aminoMsgs, 
          transaction.fee, 
          transaction.chainId, 
          transaction.memo, 
          transaction.signerData.accountNumber, 
          transaction.signerData.sequence
        )
        
        console.log('Trying amino signing for vote...')
        const aminoResult = await aminoSigner.signAmino(signerAddress, signDocAmino)
        
        // Convert amino result to direct signing format
        const signedTxBody = {
          messages: aminoResult.signed.msgs.map((msg) => aminoTypes.fromAmino(msg)),
          memo: aminoResult.signed.memo,
        }
        
        const signedTxBodyEncodeObject: TxBodyEncodeObject = {
          typeUrl: "/cosmos.tx.v1beta1.TxBody",
          value: signedTxBody,
        }
        
        const signedTxBodyBytes = registry.encode(signedTxBodyEncodeObject)
        const signedGasLimit = Number(aminoResult.signed.fee.gas)
        const signedSequence = Number(aminoResult.signed.sequence)
        
        const signedAuthInfoBytes = makeAuthInfoBytes(
          [{ pubkey, sequence: signedSequence }],
          aminoResult.signed.fee.amount,
          signedGasLimit,
          aminoResult.signed.fee.granter,
          aminoResult.signed.fee.payer,
          SignMode.SIGN_MODE_LEGACY_AMINO_JSON,
        )
        
        signed = {
          bodyBytes: signedTxBodyBytes,
          authInfoBytes: signedAuthInfoBytes,
        }
        signature = aminoResult.signature
        
        console.log('Amino signing successful for vote')
        
      } catch (aminoError) {
        console.log('Amino signing failed, trying direct signing for vote:', aminoError)
        
        // Fallback to direct signing
        const result = await offlineSigner.signDirect(signerAddress, signDoc)
        signature = result.signature
        signed = result.signed
        
        console.log('Direct signing successful for vote')
      }

      // Create final transaction
      const txRaw = TxRaw.fromPartial({
        bodyBytes: signed.bodyBytes,
        authInfoBytes: signed.authInfoBytes,
        signatures: [fromBase64(signature.signature)],
      })

      // Broadcast using ping.pub's approach - via REST API
      const txBytes = TxRaw.encode(txRaw).finish()
      const txBase64 = btoa(String.fromCharCode(...txBytes))
      
      const broadcastRequest = {
        tx_bytes: txBase64,
        mode: 'BROADCAST_MODE_SYNC'
      }

      const broadcastResponse = await fetch(`${getApiBaseUrl()}/cosmos/tx/v1beta1/txs`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(broadcastRequest)
      })

      if (!broadcastResponse.ok) {
        throw new Error('Failed to broadcast vote transaction')
      }

      const result = await broadcastResponse.json()

      console.log('Manual vote result:', result)
      
      // Check for broadcast errors
      if (result.code && result.code !== 0) {
        throw new Error(result.message || 'Broadcast error')
      }

      if (result.tx_response && result.tx_response.code !== 0) {
        throw new Error(result.tx_response.raw_log || 'Vote transaction failed')
      }

      return result.tx_response || result

    } catch (error) {
      console.error('Manual vote failed:', error)
      throw error
    }
  }

  // Manual proposal creation function following same ping.pub approach
  async function submitProposalManually(
    proposalType: 'text' | 'community-spend' | 'param-change' | 'software-upgrade' | 'cancel-upgrade',
    title: string,
    description: string,
    initialDeposit: string,
    denom: string,
    // For community spend
    recipient?: string,
    spendAmount?: string,
    // For parameter change
    paramSubspace?: string,
    paramKey?: string,
    paramValue?: string,
    // For software upgrade
    upgradeName?: string,
    upgradeHeight?: number,
    upgradeInfo?: string,
    // For cancel upgrade
    cancelUpgradeName?: string
  ): Promise<any> {
    if (!isConnected.value || !window.keplr) {
      throw new Error('Wallet not connected')
    }

    try {
      const { currentNetwork } = networkStore
      const signerAddress = address.value

      // Get account info from chain using REST API (exactly like ping.pub)
      const accountResponse = await fetch(`${getApiBaseUrl()}/cosmos/auth/v1beta1/accounts/${signerAddress}`)
      if (!accountResponse.ok) {
        throw new Error('Account not found on chain')
      }
      
      const accountData = await accountResponse.json()
      
      // Use ping.pub's findField approach to extract account_number and sequence
      const findField = (obj: any, name: string): any => {
        if (!obj) return undefined
        const list = Object.keys(obj).filter(x => x && !x.startsWith("@"))
        if (list.includes(name)) {
          return obj[name]
        }
        for (let i = 0; i < list.length; i++) {
          const field = obj[list[i]]
          if (typeof field === 'string') continue
          if (Array.isArray(field)) continue
          const sub = findField(field, name)
          if (sub) return sub
        }
        return undefined
      }

      const accountNumber = Number(findField(accountData, "account_number"))
      const sequence = Number(findField(accountData, "sequence"))

      if (accountNumber === undefined || sequence === undefined) {
        throw new Error('Could not extract account number or sequence from account data')
      }

      console.log('Account info for proposal submission:', { accountNumber, sequence })

      // Create proposal content based on type
      let proposalContent: Any;
      
      if (proposalType === 'text') {
        proposalContent = Any.fromPartial({
          typeUrl: '/cosmos.gov.v1beta1.TextProposal',
          value: TextProposal.encode(TextProposal.fromPartial({
            title,
            description
          })).finish()
        })
      } else if (proposalType === 'community-spend' && recipient && spendAmount) {
        proposalContent = Any.fromPartial({
          typeUrl: '/cosmos.distribution.v1beta1.CommunityPoolSpendProposal',
          value: CommunityPoolSpendProposal.encode(CommunityPoolSpendProposal.fromPartial({
            title,
            description,
            recipient,
            amount: [{
              denom: denom,
              amount: spendAmount
            }]
          })).finish()
        })
      } else if (proposalType === 'param-change' && paramSubspace && paramKey && paramValue) {
        proposalContent = Any.fromPartial({
          typeUrl: '/cosmos.params.v1beta1.ParameterChangeProposal',
          value: ParameterChangeProposal.encode(ParameterChangeProposal.fromPartial({
            title,
            description,
            changes: [{
              subspace: paramSubspace,
              key: paramKey,
              value: paramValue
            }]
          })).finish()
        })
      } else if (proposalType === 'software-upgrade' && upgradeName && upgradeHeight) {
        proposalContent = Any.fromPartial({
          typeUrl: '/cosmos.upgrade.v1beta1.SoftwareUpgradeProposal',
          value: SoftwareUpgradeProposal.encode(SoftwareUpgradeProposal.fromPartial({
            title,
            description,
            plan: Plan.fromPartial({
              name: upgradeName,
              height: BigInt(upgradeHeight),
              info: upgradeInfo || ''
            })
          })).finish()
        })
      } else if (proposalType === 'cancel-upgrade') {
        proposalContent = Any.fromPartial({
          typeUrl: '/cosmos.upgrade.v1beta1.CancelSoftwareUpgradeProposal',
          value: CancelSoftwareUpgradeProposal.encode(CancelSoftwareUpgradeProposal.fromPartial({
            title,
            description
          })).finish()
        })
      } else {
        throw new Error('Invalid proposal type or missing required parameters')
      }

      // Create the transaction object exactly like ping.pub does
      const transaction = {
        chainId: currentNetwork.chainId,
        signerAddress: signerAddress,
        messages: [
          {
            typeUrl: '/cosmos.gov.v1beta1.MsgSubmitProposal',
            value: {
              content: proposalContent,
              initialDeposit: [{
                denom: denom,
                amount: initialDeposit
              }],
              proposer: signerAddress
            }
          }
        ],
        fee: {
          gas: "300000", // Higher gas for proposal submission
          amount: [
            { amount: "7500", denom: denom }
          ]
        },
        memo: "",
        signerData: {
          accountNumber: accountNumber,
          sequence: sequence,
          chainId: currentNetwork.chainId
        }
      }

      console.log('Created proposal submission transaction object:', transaction)

      // Create a Keplr wallet instance like ping.pub does
      const offlineSigner: OfflineDirectSigner = window.getOfflineSigner(currentNetwork.chainId)

      // Use ping.pub's signing approach - construct the signDoc manually but let Keplr handle it
      const registry = new Registry(defaultRegistryTypes)

      // Wrap message in Any type for encoding
      const msgAny = Any.fromPartial({
        typeUrl: '/cosmos.gov.v1beta1.MsgSubmitProposal',
        value: MsgSubmitProposal.encode(MsgSubmitProposal.fromPartial(transaction.messages[0].value)).finish()
      })

      // Create transaction body
      const txBodyEncodeObject: TxBodyEncodeObject = {
        typeUrl: "/cosmos.tx.v1beta1.TxBody",
        value: {
          messages: [msgAny],
          memo: transaction.memo,
        },
      }

      const txBodyBytes = registry.encode(txBodyEncodeObject)

      // Get signer accounts for pubkey
      const accounts = await offlineSigner.getAccounts()
      if (accounts.length === 0) {
        throw new Error('No accounts found')
      }

      // Create public key with correct type for this chain
      const pubkey = Any.fromPartial({
        typeUrl: getKeyType(currentNetwork.chainId),
        value: PubKey.encode({
          key: accounts[0].pubkey,
        }).finish()
      })

      const gasLimit = Number(transaction.fee.gas)
      const authInfoBytes = makeAuthInfoBytes(
        [{ pubkey, sequence: transaction.signerData.sequence }],
        transaction.fee.amount,
        gasLimit
      )

      const signDoc = makeSignDoc(
        txBodyBytes, 
        authInfoBytes, 
        transaction.chainId, 
        transaction.signerData.accountNumber
      )

      console.log('About to sign proposal submission with Keplr:', {
        signerAddress: transaction.signerAddress,
        chainId: transaction.chainId,
        proposalType: proposalType,
        title: title,
        accountNumber: transaction.signerData.accountNumber,
        sequence: transaction.signerData.sequence,
        keyType: getKeyType(currentNetwork.chainId)
      })

      // Try amino signing first (more compatible with Ethermint chains)
      let signature: any, signed: any;
      
      try {
        // Skip amino signing for upgrade proposals since they're not supported
        if (proposalType === 'software-upgrade' || proposalType === 'cancel-upgrade') {
          throw new Error('Upgrade proposals not supported in amino, using direct signing')
        }
        
        // Try amino signing first
        const aminoTypes = new AminoTypes(createDefaultAminoConverters())
        const aminoSigner = window.getOfflineSignerOnlyAmino(currentNetwork.chainId) as OfflineAminoSigner
        
        const aminoMsgs = transaction.messages.map((msg) => aminoTypes.toAmino(msg))
        const signDocAmino = makeSignDocAmino(
          aminoMsgs, 
          transaction.fee, 
          transaction.chainId, 
          transaction.memo, 
          transaction.signerData.accountNumber, 
          transaction.signerData.sequence
        )
        
        console.log('Trying amino signing for proposal submission...')
        const aminoResult = await aminoSigner.signAmino(signerAddress, signDocAmino)
        
        // Convert amino result to direct signing format
        const signedTxBody = {
          messages: aminoResult.signed.msgs.map((msg) => aminoTypes.fromAmino(msg)),
          memo: aminoResult.signed.memo,
        }
        
        const signedTxBodyEncodeObject: TxBodyEncodeObject = {
          typeUrl: "/cosmos.tx.v1beta1.TxBody",
          value: signedTxBody,
        }
        
        const signedTxBodyBytes = registry.encode(signedTxBodyEncodeObject)
        const signedGasLimit = Number(aminoResult.signed.fee.gas)
        const signedSequence = Number(aminoResult.signed.sequence)
        
        const signedAuthInfoBytes = makeAuthInfoBytes(
          [{ pubkey, sequence: signedSequence }],
          aminoResult.signed.fee.amount,
          signedGasLimit,
          aminoResult.signed.fee.granter,
          aminoResult.signed.fee.payer,
          SignMode.SIGN_MODE_LEGACY_AMINO_JSON,
        )
        
        signed = {
          bodyBytes: signedTxBodyBytes,
          authInfoBytes: signedAuthInfoBytes,
        }
        signature = aminoResult.signature
        
        console.log('Amino signing successful for proposal submission')
        
      } catch (aminoError) {
        console.log('Amino signing failed, trying direct signing for proposal submission:', aminoError)
        
        // Fallback to direct signing
        const result = await offlineSigner.signDirect(signerAddress, signDoc)
        signature = result.signature
        signed = result.signed
        
        console.log('Direct signing successful for proposal submission')
      }

      // Create final transaction
      const txRaw = TxRaw.fromPartial({
        bodyBytes: signed.bodyBytes,
        authInfoBytes: signed.authInfoBytes,
        signatures: [fromBase64(signature.signature)],
      })

      // Broadcast using ping.pub's approach - via REST API
      const txBytes = TxRaw.encode(txRaw).finish()
      const txBase64 = btoa(String.fromCharCode(...txBytes))
      
      const broadcastRequest = {
        tx_bytes: txBase64,
        mode: 'BROADCAST_MODE_SYNC'
      }

      console.log('Broadcasting proposal submission...')
      const broadcastResponse = await fetch(`${getApiBaseUrl()}/cosmos/tx/v1beta1/txs`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(broadcastRequest)
      })

      if (!broadcastResponse.ok) {
        throw new Error('Failed to broadcast proposal submission transaction')
      }

      const result = await broadcastResponse.json()

      console.log('Manual proposal submission result:', result)
      
      // Check for broadcast errors
      if (result.code && result.code !== 0) {
        throw new Error(result.message || 'Broadcast error')
      }

      if (result.tx_response && result.tx_response.code !== 0) {
        console.error('Transaction failed with code:', result.tx_response.code)
        console.error('Raw log:', result.tx_response.raw_log)
        throw new Error(result.tx_response.raw_log || 'Proposal submission transaction failed')
      }

      return result.tx_response || result

    } catch (error) {
      console.error('Manual proposal submission failed:', error)
      throw error
    }
  }

  // Manual claim rewards function following same ping.pub approach
  async function claimRewardsManually(
    validatorAddress?: string
  ): Promise<any> {
    if (!isConnected.value || !window.keplr) {
      throw new Error('Wallet not connected')
    }

    try {
      const { currentNetwork } = networkStore
      const signerAddress = address.value

      // Get account info from chain using REST API (exactly like ping.pub)
      const accountResponse = await fetch(`${getApiBaseUrl()}/cosmos/auth/v1beta1/accounts/${signerAddress}`)
      if (!accountResponse.ok) {
        throw new Error('Account not found on chain')
      }
      
      const accountData = await accountResponse.json()
      
      // Use ping.pub's findField approach to extract account_number and sequence
      const findField = (obj: any, name: string): any => {
        if (!obj) return undefined
        const list = Object.keys(obj).filter(x => x && !x.startsWith("@"))
        if (list.includes(name)) {
          return obj[name]
        }
        for (let i = 0; i < list.length; i++) {
          const field = obj[list[i]]
          if (typeof field === 'string') continue
          if (Array.isArray(field)) continue
          const sub = findField(field, name)
          if (sub) return sub
        }
        return undefined
      }

      const accountNumber = Number(findField(accountData, "account_number"))
      const sequence = Number(findField(accountData, "sequence"))

      if (accountNumber === undefined || sequence === undefined) {
        throw new Error('Could not extract account number or sequence from account data')
      }

      console.log('Account info for manual claim rewards:', { accountNumber, sequence })

      // Create the transaction object based on whether claiming from specific validator or all
      let messages: any[]
      
      if (validatorAddress) {
        // Claim from specific validator
        messages = [{
          typeUrl: '/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward',
          value: {
            delegatorAddress: signerAddress,
            validatorAddress: validatorAddress
          }
        }]
      } else {
        // Claim from all validators - first get delegations
        const delegationsResponse = await fetch(`${getApiBaseUrl()}/cosmos/staking/v1beta1/delegations/${signerAddress}`)
        if (!delegationsResponse.ok) {
          throw new Error('Failed to fetch delegations')
        }
        
        const delegationsData = await delegationsResponse.json()
        const delegations = delegationsData.delegation_responses || []
        
        if (delegations.length === 0) {
          throw new Error('No delegations found')
        }
        
        messages = delegations.map((delegation: any) => ({
          typeUrl: '/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward',
          value: {
            delegatorAddress: signerAddress,
            validatorAddress: delegation.delegation.validator_address
          }
        }))
      }

      const transaction = {
        chainId: currentNetwork.chainId,
        signerAddress: signerAddress,
        messages: messages,
        fee: {
          gas: (200000 * messages.length).toString(),
          amount: [
            { amount: (5000 * messages.length).toString(), denom: currentNetwork.baseDenom }
          ]
        },
        memo: "",
        signerData: {
          accountNumber: accountNumber,
          sequence: sequence,
          chainId: currentNetwork.chainId
        }
      }

      console.log('Created claim rewards transaction object:', transaction)

      // Create a Keplr wallet instance like ping.pub does
      const offlineSigner: OfflineDirectSigner = window.getOfflineSigner(currentNetwork.chainId)

      // Use ping.pub's signing approach - construct the signDoc manually but let Keplr handle it
      const registry = new Registry(defaultRegistryTypes)

      // Import the MsgWithdrawDelegatorReward type
      const { MsgWithdrawDelegatorReward } = await import('cosmjs-types/cosmos/distribution/v1beta1/tx')

      // Wrap messages in Any type for encoding
      const msgAnys = messages.map(msg => Any.fromPartial({
        typeUrl: '/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward',
        value: MsgWithdrawDelegatorReward.encode(MsgWithdrawDelegatorReward.fromPartial(msg.value)).finish()
      }))

      // Create transaction body
      const txBodyEncodeObject: TxBodyEncodeObject = {
        typeUrl: "/cosmos.tx.v1beta1.TxBody",
        value: {
          messages: msgAnys,
          memo: transaction.memo,
        },
      }

      const txBodyBytes = registry.encode(txBodyEncodeObject)

      // Get signer accounts for pubkey
      const accounts = await offlineSigner.getAccounts()
      if (accounts.length === 0) {
        throw new Error('No accounts found')
      }

      // Create public key with correct type for this chain
      const pubkey = Any.fromPartial({
        typeUrl: getKeyType(currentNetwork.chainId),
        value: PubKey.encode({
          key: accounts[0].pubkey,
        }).finish()
      })

      const gasLimit = Number(transaction.fee.gas)
      const authInfoBytes = makeAuthInfoBytes(
        [{ pubkey, sequence: transaction.signerData.sequence }],
        transaction.fee.amount,
        gasLimit
      )

      const signDoc = makeSignDoc(
        txBodyBytes, 
        authInfoBytes, 
        transaction.chainId, 
        transaction.signerData.accountNumber
      )

      console.log('About to sign claim rewards with Keplr:', {
        signerAddress: transaction.signerAddress,
        chainId: transaction.chainId,
        accountNumber: transaction.signerData.accountNumber,
        sequence: transaction.signerData.sequence,
        keyType: getKeyType(currentNetwork.chainId),
        messageCount: messages.length
      })

      // Try amino signing first (more compatible with Ethermint chains)
      let signature: any, signed: any;
      
      try {
        // Try amino signing first
        const aminoTypes = new AminoTypes(createDefaultAminoConverters())
        const aminoSigner = window.getOfflineSignerOnlyAmino(currentNetwork.chainId) as OfflineAminoSigner
        
        const aminoMsgs = transaction.messages.map((msg) => aminoTypes.toAmino(msg))
        const signDocAmino = makeSignDocAmino(
          aminoMsgs, 
          transaction.fee, 
          transaction.chainId, 
          transaction.memo, 
          transaction.signerData.accountNumber, 
          transaction.signerData.sequence
        )
        
        console.log('Trying amino signing for claim rewards...')
        const aminoResult = await aminoSigner.signAmino(signerAddress, signDocAmino)
        
        // Convert amino result to direct signing format
        const signedTxBody = {
          messages: aminoResult.signed.msgs.map((msg) => aminoTypes.fromAmino(msg)),
          memo: aminoResult.signed.memo,
        }
        
        const signedTxBodyEncodeObject: TxBodyEncodeObject = {
          typeUrl: "/cosmos.tx.v1beta1.TxBody",
          value: signedTxBody,
        }
        
        const signedTxBodyBytes = registry.encode(signedTxBodyEncodeObject)
        const signedGasLimit = Number(aminoResult.signed.fee.gas)
        const signedSequence = Number(aminoResult.signed.sequence)
        
        const signedAuthInfoBytes = makeAuthInfoBytes(
          [{ pubkey, sequence: signedSequence }],
          aminoResult.signed.fee.amount,
          signedGasLimit,
          aminoResult.signed.fee.granter,
          aminoResult.signed.fee.payer,
          SignMode.SIGN_MODE_LEGACY_AMINO_JSON,
        )
        
        signed = {
          bodyBytes: signedTxBodyBytes,
          authInfoBytes: signedAuthInfoBytes,
        }
        signature = aminoResult.signature
        
        console.log('Amino signing successful for claim rewards')
        
      } catch (aminoError) {
        console.log('Amino signing failed, trying direct signing for claim rewards:', aminoError)
        
        // Fallback to direct signing
        const result = await offlineSigner.signDirect(signerAddress, signDoc)
        signature = result.signature
        signed = result.signed
        
        console.log('Direct signing successful for claim rewards')
      }

      // Create final transaction
      const txRaw = TxRaw.fromPartial({
        bodyBytes: signed.bodyBytes,
        authInfoBytes: signed.authInfoBytes,
        signatures: [fromBase64(signature.signature)],
      })

      // Broadcast using ping.pub's approach - via REST API
      const txBytes = TxRaw.encode(txRaw).finish()
      const txBase64 = btoa(String.fromCharCode(...txBytes))
      
      const broadcastRequest = {
        tx_bytes: txBase64,
        mode: 'BROADCAST_MODE_SYNC'
      }

      const broadcastResponse = await fetch(`${getApiBaseUrl()}/cosmos/tx/v1beta1/txs`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(broadcastRequest)
      })

      if (!broadcastResponse.ok) {
        throw new Error('Failed to broadcast claim rewards transaction')
      }

      const result = await broadcastResponse.json()

      console.log('Manual claim rewards result:', result)
      
      // Check for broadcast errors
      if (result.code && result.code !== 0) {
        throw new Error(result.message || 'Broadcast error')
      }

      if (result.tx_response && result.tx_response.code !== 0) {
        throw new Error(result.tx_response.raw_log || 'Claim rewards transaction failed')
      }

      return result.tx_response || result

    } catch (error) {
      console.error('Manual claim rewards failed:', error)
      throw error
    }
  }

  function disconnect() {
    isConnected.value = false
    address.value = ''
    balance.value = '0'
    client.value = null
  }
  
  // Format balance for display
  function formatBalance(amount: string = balance.value, decimals: number = 18): string {
    const num = parseFloat(amount) / Math.pow(10, decimals)
    return num.toLocaleString('en-US', { 
      minimumFractionDigits: 0, 
      maximumFractionDigits: 6 
    })
  }
  
  return {
    isConnected,
    address,
    shortAddress,
    balance,
    isConnecting,
    client,
    connectKeplr,
    updateBalance,
    getSigningClient,
    delegateTokensManually,
    undelegateTokensManually,
    claimRewardsManually,
    voteOnProposalManually,
    submitProposalManually,
    disconnect,
    formatBalance
  }
})