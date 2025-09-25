import { useNetworkStore } from '@/stores/network'
import { SigningStargateClient } from '@cosmjs/stargate'
import { TxRaw } from 'cosmjs-types/cosmos/tx/v1beta1/tx'
import { DeliverTxResponse, GasPrice } from '@cosmjs/stargate'

export class TransactionService {
  private getBaseUrl(): string {
    const networkStore = useNetworkStore()
    return networkStore.currentNetwork.apiEndpoint
  }

  async broadcastTx(txRaw: TxRaw): Promise<DeliverTxResponse> {
    const baseUrl = this.getBaseUrl()
    const url = `${baseUrl}/cosmos/tx/v1beta1/txs`
    
    try {
      const txBytes = TxRaw.encode(txRaw).finish()
      const txBytesBase64 = Buffer.from(txBytes).toString('base64')
      
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          tx_bytes: txBytesBase64,
          mode: 'BROADCAST_MODE_SYNC'
        })
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const result = await response.json()
      
      // Convert REST API response to DeliverTxResponse format
      return {
        code: parseInt(result.tx_response?.code || '0'),
        height: parseInt(result.tx_response?.height || '0'),
        rawLog: result.tx_response?.raw_log || '',
        transactionHash: result.tx_response?.txhash || '',
        gasUsed: parseInt(result.tx_response?.gas_used || '0'),
        gasWanted: parseInt(result.tx_response?.gas_wanted || '0'),
        events: result.tx_response?.events || []
      }
    } catch (error) {
      console.error('Failed to broadcast transaction:', error)
      throw error
    }
  }

  // Simplified delegation function that uses offline signing + REST broadcast
  async delegateTokens(
    client: SigningStargateClient,
    delegatorAddress: string,
    validatorAddress: string,
    amount: { denom: string; amount: string },
    fee: any
  ): Promise<DeliverTxResponse> {
    try {
      // For now, we'll show an alert that this functionality requires RPC access
      // In a full implementation, you would:
      // 1. Create the delegation message
      // 2. Sign it with the offline client
      // 3. Broadcast via REST API
      
      throw new Error('Transaction broadcasting via REST API not fully implemented yet. This requires RPC endpoint access which has CORS restrictions.')
    } catch (error) {
      throw error
    }
  }
}

export const transactionService = new TransactionService()