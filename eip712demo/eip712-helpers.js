/**
 * Example: Creating EIP-712 TypedData for Cosmos Transactions on Shardeum
 * 
 * This module demonstrates how to construct EIP-712 typed data for various
 * Cosmos SDK messages to be signed by MetaMask.
 */

// ============================================================================
// Configuration
// ============================================================================

const CONFIG = {
  chainId: 8117,              // EVM Chain ID (8117 for local, 8118 for mainnet, 8119 for testnet)
  cosmosChainId: 'shardeum_8117-1',  // Cosmos Chain ID
  denom: 'ashm',              // Base denomination
  gasLimit: '200000',
  feeAmount: '2000',
  bech32Prefix: 'shardeum',
  validatorPrefix: 'shardeumvaloper',
};

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Convert Ethereum hex address to Bech32 format
 * Note: In production, use @cosmjs/encoding library
 */
function hexToBech32(hexAddress, prefix = CONFIG.bech32Prefix) {
  // Remove 0x prefix
  const hex = hexAddress.startsWith('0x') ? hexAddress.slice(2) : hexAddress;
  
  // This is a simplified version - use proper Bech32 encoding in production
  // For now, we'll use a placeholder format with the full hex address
  return `${prefix}1${hex.toLowerCase()}`;
}

/**
 * Query account information from the chain
 */
async function getAccountInfo(address, restUrl = 'http://localhost:1317') {
  try {
    const response = await fetch(
      `${restUrl}/cosmos/auth/v1beta1/accounts/${address}`
    );
    const data = await response.json();
    
    return {
      accountNumber: data.account.account_number || '0',
      sequence: data.account.sequence || '0',
    };
  } catch (error) {
    console.warn('Could not query account info, using defaults:', error.message);
    return {
      accountNumber: '0',
      sequence: '0',
    };
  }
}

// ============================================================================
// EIP-712 TypedData Builders
// ============================================================================

/**
 * Build base EIP-712 types structure
 * These types are required for all Cosmos transactions
 */
function getBaseTypes() {
  return {
    EIP712Domain: [
      { name: 'name', type: 'string' },
      { name: 'version', type: 'string' },
      { name: 'chainId', type: 'uint256' },
      { name: 'verifyingContract', type: 'string' },
      { name: 'salt', type: 'string' },
    ],
    Tx: [
      { name: 'account_number', type: 'string' },
      { name: 'chain_id', type: 'string' },
      { name: 'fee', type: 'Fee' },
      { name: 'memo', type: 'string' },
      { name: 'msgs', type: 'Msg[]' },
      { name: 'sequence', type: 'string' },
    ],
    Fee: [
      { name: 'amount', type: 'Coin[]' },
      { name: 'gas', type: 'string' },
      // Note: feePayer removed - using standard Cosmos fee structure without delegation
    ],
    Coin: [
      { name: 'denom', type: 'string' },
      { name: 'amount', type: 'string' },
    ],
    Msg: [
      { name: 'type', type: 'string' },
      { name: 'value', type: 'MsgValue' },
    ],
  };
}

/**
 * Build domain for EIP-712
 */
function getDomain(chainId = CONFIG.chainId) {
  return {
    name: 'Cosmos Web3',
    version: '1.0.0',
    chainId: '0x' + chainId.toString(16), // Convert to hex format (e.g., 8117 → "0x1fb5")
    verifyingContract: 'cosmos',
    salt: '0',
  };
}

// ============================================================================
// Message Type Builders
// ============================================================================

/**
 * Build EIP-712 TypedData for MsgDelegate (Staking)
 */
async function buildMsgDelegate(params) {
  const {
    delegatorAddress,    // Ethereum hex address (0x...)
    validatorAddress,    // Bech32 validator address (shardeumvaloper...)
    amount,              // Amount in base units (e.g., "1000000")
    memo = '',
    restUrl,
  } = params;

  // Convert Ethereum address to Bech32
  const delegatorBech32 = hexToBech32(delegatorAddress);

  // Get account info
  const accountInfo = await getAccountInfo(delegatorBech32, restUrl);

  // Build types
  const types = {
    ...getBaseTypes(),
    MsgValue: [
      { name: 'delegator_address', type: 'string' },
      { name: 'validator_address', type: 'string' },
      { name: 'amount', type: 'TypeAmount' },
    ],
    TypeAmount: [
      { name: 'denom', type: 'string' },
      { name: 'amount', type: 'string' },
    ],
  };

  // Build message
  const message = {
    account_number: accountInfo.accountNumber,
    chain_id: CONFIG.cosmosChainId,
    fee: {
      amount: [{
        denom: CONFIG.denom,
        amount: CONFIG.feeAmount,
      }],
      gas: CONFIG.gasLimit,
    },
    memo: memo,
    msgs: [{
      type: 'cosmos-sdk/MsgDelegate',
      value: {
        delegator_address: delegatorBech32,
        validator_address: validatorAddress,
        amount: {
          denom: CONFIG.denom,
          amount: amount,
        },
      },
    }],
    sequence: accountInfo.sequence,
  };

  return {
    types,
    primaryType: 'Tx',
    domain: getDomain(),
    message,
  };
}

/**
 * Build EIP-712 TypedData for MsgSend (Transfer)
 */
async function buildMsgSend(params) {
  const {
    fromAddress,
    toAddress,
    amount,
    memo = '',
    restUrl,
  } = params;

  const fromBech32 = hexToBech32(fromAddress);
  const toBech32 = hexToBech32(toAddress);
  const accountInfo = await getAccountInfo(fromBech32, restUrl);

  const types = {
    ...getBaseTypes(),
    MsgValue: [
      { name: 'from_address', type: 'string' },
      { name: 'to_address', type: 'string' },
      { name: 'amount', type: 'TypeAmount[]' },
    ],
    TypeAmount: [
      { name: 'denom', type: 'string' },
      { name: 'amount', type: 'string' },
    ],
  };

  const message = {
    account_number: accountInfo.accountNumber,
    chain_id: CONFIG.cosmosChainId,
    fee: {
      amount: [{
        denom: CONFIG.denom,
        amount: CONFIG.feeAmount,
      }],
      gas: CONFIG.gasLimit,
    },
    memo: memo,
    msgs: [{
      type: 'cosmos-sdk/MsgSend',
      value: {
        from_address: fromBech32,
        to_address: toBech32,
        amount: [{
          denom: CONFIG.denom,
          amount: amount,
        }],
      },
    }],
    sequence: accountInfo.sequence,
  };

  return {
    types,
    primaryType: 'Tx',
    domain: getDomain(),
    message,
  };
}

/**
 * Build EIP-712 TypedData for MsgWithdrawDelegatorReward
 */
async function buildMsgWithdrawReward(params) {
  const {
    delegatorAddress,
    validatorAddress,
    memo = '',
    restUrl,
  } = params;

  const delegatorBech32 = hexToBech32(delegatorAddress);
  const accountInfo = await getAccountInfo(delegatorBech32, restUrl);

  const types = {
    ...getBaseTypes(),
    MsgValue: [
      { name: 'delegator_address', type: 'string' },
      { name: 'validator_address', type: 'string' },
    ],
  };

  const message = {
    account_number: accountInfo.accountNumber,
    chain_id: CONFIG.cosmosChainId,
    fee: {
      amount: [{
        denom: CONFIG.denom,
        amount: CONFIG.feeAmount,
      }],
      gas: CONFIG.gasLimit,
    },
    memo: memo,
    msgs: [{
      type: 'cosmos-sdk/MsgWithdrawDelegationReward',
      value: {
        delegator_address: delegatorBech32,
        validator_address: validatorAddress,
      },
    }],
    sequence: accountInfo.sequence,
  };

  return {
    types,
    primaryType: 'Tx',
    domain: getDomain(),
    message,
  };
}

/**
 * Build EIP-712 TypedData for MsgVote (Governance)
 */
async function buildMsgVote(params) {
  const {
    voterAddress,
    proposalId,
    option,        // 1=Yes, 2=Abstain, 3=No, 4=NoWithVeto
    memo = '',
    restUrl,
  } = params;

  const voterBech32 = hexToBech32(voterAddress);
  const accountInfo = await getAccountInfo(voterBech32, restUrl);

  const types = {
    ...getBaseTypes(),
    MsgValue: [
      { name: 'proposal_id', type: 'string' },
      { name: 'voter', type: 'string' },
      { name: 'option', type: 'int32' },
    ],
  };

  const message = {
    account_number: accountInfo.accountNumber,
    chain_id: CONFIG.cosmosChainId,
    fee: {
      amount: [{
        denom: CONFIG.denom,
        amount: CONFIG.feeAmount,
      }],
      gas: CONFIG.gasLimit,
    },
    memo: memo,
    msgs: [{
      type: 'cosmos-sdk/MsgVote',
      value: {
        proposal_id: proposalId.toString(),
        voter: voterBech32,
        option: option,
      },
    }],
    sequence: accountInfo.sequence,
  };

  return {
    types,
    primaryType: 'Tx',
    domain: getDomain(),
    message,
  };
}

// ============================================================================
// Signing and Broadcasting
// ============================================================================

/**
 * Sign TypedData with MetaMask
 */
async function signWithMetaMask(account, typedData) {
  if (!window.ethereum) {
    throw new Error('MetaMask not found');
  }

  const signature = await window.ethereum.request({
    method: 'eth_signTypedData_v4',
    params: [account, JSON.stringify(typedData)],
  });

  return signature;
}

/**
 * Broadcast signed transaction to the network
 * Note: This is a simplified version - full implementation requires
 * proper transaction encoding with the signature
 */
async function broadcastTransaction(signedTx, rpcUrl = 'http://localhost:1317') {
  // In a real implementation, you would:
  // 1. Construct the full Cosmos transaction with the signature
  // 2. Encode it properly
  // 3. Send to /cosmos/tx/v1beta1/txs endpoint
  
  const response = await fetch(`${rpcUrl}/cosmos/tx/v1beta1/txs`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      tx_bytes: signedTx,
      mode: 'BROADCAST_MODE_SYNC',
    }),
  });

  return await response.json();
}

// ============================================================================
// Usage Examples
// ============================================================================

/**
 * Example 1: Delegate Stake
 */
async function exampleDelegate() {
  // Get account from MetaMask
  const accounts = await window.ethereum.request({ 
    method: 'eth_requestAccounts' 
  });
  const account = accounts[0];

  // Build TypedData
  const typedData = await buildMsgDelegate({
    delegatorAddress: account,
    validatorAddress: 'shardeumvaloper1abc...xyz',
    amount: '1000000',  // 1 SHM = 1000000 ashm
    memo: 'Delegating via MetaMask',
  });

  console.log('TypedData:', JSON.stringify(typedData, null, 2));

  // Sign with MetaMask
  const signature = await signWithMetaMask(account, typedData);
  console.log('Signature:', signature);

  return signature;
}

/**
 * Example 2: Send Tokens
 */
async function exampleSend() {
  const accounts = await window.ethereum.request({ 
    method: 'eth_requestAccounts' 
  });
  const account = accounts[0];

  const typedData = await buildMsgSend({
    fromAddress: account,
    toAddress: '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb',
    amount: '1000000',
    memo: 'Payment via MetaMask',
  });

  const signature = await signWithMetaMask(account, typedData);
  return signature;
}

/**
 * Example 3: Withdraw Rewards
 */
async function exampleWithdrawRewards() {
  const accounts = await window.ethereum.request({ 
    method: 'eth_requestAccounts' 
  });
  const account = accounts[0];

  const typedData = await buildMsgWithdrawReward({
    delegatorAddress: account,
    validatorAddress: 'shardeumvaloper1abc...xyz',
    memo: 'Claiming rewards',
  });

  const signature = await signWithMetaMask(account, typedData);
  return signature;
}

// ============================================================================
// Export functions
// ============================================================================

if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    CONFIG,
    buildMsgDelegate,
    buildMsgSend,
    buildMsgWithdrawReward,
    buildMsgVote,
    signWithMetaMask,
    broadcastTransaction,
    hexToBech32,
    getAccountInfo,
  };
}

// ============================================================================
// Browser console helpers
// ============================================================================

if (typeof window !== 'undefined') {
  window.ShardeumEIP712 = {
    buildMsgDelegate,
    buildMsgSend,
    buildMsgWithdrawReward,
    buildMsgVote,
    signWithMetaMask,
    exampleDelegate,
    exampleSend,
    exampleWithdrawRewards,
    CONFIG,
  };
  
  console.log('🚀 Shardeum EIP-712 Helpers loaded!');
  console.log('Available functions:', Object.keys(window.ShardeumEIP712));
  console.log('Try: ShardeumEIP712.exampleDelegate()');
}
