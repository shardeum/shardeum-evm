# Shardeum Dashboard

A comprehensive web dashboard for your local Shardeum Cosmos network, inspired by ping.pub. This dashboard allows you to explore blocks, manage staking, participate in governance, and connect with Keplr wallet.

## Features

- 🔗 **Keplr Wallet Integration** - Connect and manage your wallet
- 📊 **Network Overview** - Real-time network statistics and metrics
- 👥 **Validator Management** - View and interact with network validators  
- 💰 **Staking Interface** - Delegate, undelegate, and claim rewards
- 🗳️ **Governance** - Create proposals and vote on network changes
- 🔍 **Block Explorer** - Browse blocks and transactions
- 🌐 **Multi-Network Support** - Switch between local, testnet, and devnet

## Prerequisites

Before running the dashboard, ensure you have:

1. **Shardeum Network Running**: Your local Shardeum network must be running with at least one full node
2. **Keplr Wallet**: Install the [Keplr browser extension](https://wallet.keplr.app/)
3. **Node.js**: Version 18+ recommended

## Quick Start

### 1. Start Your Shardeum Network

First, ensure your Shardeum network is running. From your project root:

```bash
# Start a local network with 4 nodes
make start-network NETWORK=local

# Or start with custom configuration
make start-network NETWORK=testnet NODES=6
```

Your network will be available on:
- **Node0 (Validator)**: RPC: `http://localhost:26657`, API: `http://localhost:1317`
- **Node1+ (Full Nodes)**: RPC: `http://localhost:26658+`, API: `http://localhost:1318+`, JSON-RPC: `http://localhost:8547+`

### 2. Install Dependencies

```bash
cd shardeum-dashboard
npm install
```

### 3. Start the Dashboard

```bash
npm run dev
```

The dashboard will be available at `http://localhost:3000`

## Network Configuration

The dashboard automatically connects to your local network endpoints:

- **Local Network**: 
  - RPC: `http://localhost:26658` (Node1 - full node)
  - API: `http://localhost:1318`
  - JSON-RPC: `http://localhost:8547`
  - Chain ID: `shardeum-local`

- **Testnet/Devnet**: Similar configuration with testnet/devnet chain IDs

## Usage Guide

### Connecting Your Wallet

1. Click "Connect Keplr" in the top right
2. Keplr will prompt you to add the Shardeum network
3. Approve the network addition and connection

### Staking Tokens

1. Navigate to the **Staking** page
2. View your current delegations and rewards
3. Click "Delegate More" or "Start Staking" to delegate to validators
4. Use "Undelegate" to unbond tokens (21-day unbonding period)
5. Click "Claim All" to claim staking rewards

### Governance Participation

1. Go to the **Governance** page
2. View active and past proposals
3. Click on a proposal to see details and vote
4. Create new proposals using "New Proposal" (requires deposit)

### Exploring the Network

1. **Dashboard**: Overview of network statistics and recent activity
2. **Validators**: Browse all validators, their voting power, and commission
3. **Blocks**: Search and explore blocks and transactions

## Development

### Project Structure

```
shardeum-dashboard/
├── src/
│   ├── components/          # Reusable Vue components
│   │   ├── NetworkSelector.vue
│   │   ├── WalletConnect.vue
│   │   └── ...
│   ├── views/               # Page components
│   │   ├── Dashboard.vue
│   │   ├── Staking.vue
│   │   ├── Governance.vue
│   │   └── ...
│   ├── stores/              # Pinia state management
│   │   ├── network.ts
│   │   └── wallet.ts
│   ├── types/               # TypeScript type definitions
│   └── router/              # Vue Router configuration
├── public/                  # Static assets
└── ...config files
```

### Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint
- `npm run type-check` - Run TypeScript checks

### Tech Stack

- **Vue 3** - Progressive JavaScript framework
- **TypeScript** - Type safety and better DX
- **Tailwind CSS** - Utility-first CSS framework
- **Vite** - Fast build tool and dev server
- **Pinia** - State management
- **CosmJS** - Cosmos SDK client library
- **Keplr Integration** - Wallet connectivity

## Network Configuration

The dashboard reads your network configuration from the same environment variables used by your Shardeum scripts:

- `SHARDEUM_NETWORK` - Network name (local, testnet, devnet, mainnet)
- `SHARDEUM_CONFIG_DIR` - Path to network configurations

Supported networks are automatically detected based on your running nodes.

## Troubleshooting

### Wallet Connection Issues

- Ensure Keplr extension is installed and unlocked
- Clear browser cache if connection fails
- Check that your network is running on the expected ports

### Network Connection Errors

- Verify your Shardeum network is running: `ps aux | grep shardeumd`
- Check node logs: `tail -f ./.[network]/node*/node.log`
- Ensure firewall isn't blocking local ports (26657-26660, 1317-1320, 8545-8550)

### Transaction Failures

- Ensure you have sufficient balance for transaction fees
- Check that your wallet is connected to the correct network
- Verify the validator address is correct for staking operations

## Advanced Configuration

### Custom Network Endpoints

Edit `src/types/network.ts` to customize network endpoints:

```typescript
export const NETWORK_CONFIGS: Record<string, NetworkConfig> = {
  local: {
    // ... customize endpoints
    rpcEndpoint: 'http://your-custom-rpc:26657',
    apiEndpoint: 'http://your-custom-api:1317'
  }
}
```

### Adding New Networks

1. Add network configuration to `NETWORK_CONFIGS`
2. Update network selector component
3. Test connection and functionality

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly with your local network
5. Submit a pull request

## License

This project is licensed under the same terms as the parent Shardeum project.

## Support

For issues and questions:
- Check your local network is running properly
- Review the troubleshooting section
- Check the browser console for errors
- Ensure Keplr wallet is properly configured