export interface NetworkConfig {
  id: string
  name: string
  chainId: string
  evmChainId: number
  rpcEndpoint: string
  apiEndpoint: string
  jsonRpcEndpoint?: string
  wsEndpoint?: string
  baseDenom: string
  decimals: number
  symbol: string
  bech32Prefix: string
}

export interface NodeEndpoints {
  rpc: string
  api: string
  jsonRpc?: string
  websocket?: string
}

export const NETWORK_CONFIGS: Record<string, NetworkConfig> = {
  local: {
    id: 'local',
    name: 'Local Network',
    chainId: 'shardeum-local',
    evmChainId: 8119,
    rpcEndpoint: 'http://localhost:26657', // Node0 Tendermint RPC endpoint
    apiEndpoint: 'http://localhost:1317',
    jsonRpcEndpoint: 'http://localhost:8547',
    wsEndpoint: 'ws://localhost:8548',
    baseDenom: 'ashm',
    decimals: 18,
    symbol: 'SHM',
    bech32Prefix: 'shardeum'
  },
  testnet: {
    id: 'testnet',
    name: 'Shardeum Testnet',
    chainId: 'shardeum-testnet',
    evmChainId: 8119,
    rpcEndpoint: 'http://localhost:26658',
    apiEndpoint: 'http://localhost:1318',
    jsonRpcEndpoint: 'http://localhost:8547',
    wsEndpoint: 'ws://localhost:8548',
    baseDenom: 'ashm',
    decimals: 18,
    symbol: 'SHM',
    bech32Prefix: 'shardeum'
  },
  devnet: {
    id: 'devnet',
    name: 'Shardeum Devnet',
    chainId: 'shardeum-devnet',
    evmChainId: 8119,
    rpcEndpoint: 'http://localhost:26658',
    apiEndpoint: 'http://localhost:1318',
    jsonRpcEndpoint: 'http://localhost:8547',
    wsEndpoint: 'ws://localhost:8548',
    baseDenom: 'ashm',
    decimals: 18,
    symbol: 'SHM',
    bech32Prefix: 'shardeum'
  }
}