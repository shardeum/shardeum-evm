#!/bin/sh

set -e

for ENVAR in PEERS SHARDEUM_NETWORK; do
  eval value=\$$ENVAR
  if [ -z "$value" ]; then
    echo "Missing required env variable $ENVAR"
    missing=1
  fi
done

MONIKER=${MONIKER:-docker}

case $NODE_TYPE in
  API)
    NODE_HOME="/app"
    ;;
  SENTRY)
    NODE_HOME="/app/.shardeumd"
    ;;
esac

if [ $missing ]; then exit 1; fi

if [ ! -e "$NODE_HOME/data/state.db" ]; then
  /app/shardeumd init $MONIKER --home $NODE_HOME
fi

if [ "$NODE_TYPE" = "SENTRY" ]; then
    ###
    # set max_num_inbound_peers to 80
    # ---
    sed -i 's/^max_num_inbound_peers = .*/max_num_inbound_peers = 80/' "$NODE_HOME/config/config.toml"
    cat $NODE_HOME/config/config.toml | grep max_num_inbound_peers

    ###
    # set max_num_outbound_peers to 30
    # ---
    sed -i 's/^max_num_outbound_peers = .*/max_num_outbound_peers = 30/' "$NODE_HOME/config/config.toml"
    cat $NODE_HOME/config/config.toml | grep max_num_outbound_peers

    ###
    # set send_rate to 10 MB
    # ---
    sed -i 's/^send_rate = .*/send_rate = 10240000/' "$NODE_HOME/config/config.toml"
    cat $NODE_HOME/config/config.toml | grep send_rate

    ###
    # set recv_rate to 10 MB
    # ---
    sed -i 's/^recv_rate = .*/recv_rate = 10240000/' "$NODE_HOME/config/config.toml"
    cat $NODE_HOME/config/config.toml | grep recv_rate

    ###
    # set size to 10000 (Larger for relay)
    # ---
    sed -i 's/^size = .*/size = 10000/' "$NODE_HOME/config/config.toml"
    cat $NODE_HOME/config/config.toml | grep size

    ###
    # set broadcast to true, by default it's true probably (Must broadcast)
    # ---
    sed -i 's/^broadcast = .*/broadcast = true/' "$NODE_HOME/config/config.toml"
    cat $NODE_HOME/config/config.toml | grep broadcast

    ###
    # set rpc.laddr as "" (this isn't working as this will start rpc on 127.0.0.1:DEFAULT_PORT even though laddr becomes set to empty like "")
    # ---
    sed -i '/^\[rpc\]/,/^\[/{s/^laddr *=.*/laddr = ""/}' "$NODE_HOME/config/config.toml"
    grep -A5 "\[rpc\]" $NODE_HOME/config/config.toml

fi

case $SHARDEUM_NETWORK in
  local)
    CHAIN_ID=shardeum_8117-1
    EVM_CHAIN_ID=8117
    # cp /app/config/local-genesis.json $NODE_HOME/config/genesis.json
    echo "Downloading genesis from to =====> $NODE_HOME/config/genesis.json"
    # wget $GENESIS_URL -O $NODE_HOME/config/genesis.json
    curl -s $GENESIS_URL | jq -r '.result.genesis' > $NODE_HOME/config/genesis.json
    ;;
  testnet)
    CHAIN_ID=shardeum-8119-2
    EVM_CHAIN_ID=8119
    cp /app/config/testnet-genesis.json $NODE_HOME/config/genesis.json
    ;;
  mainnet)
    CHAIN_ID=shardeum_8118-1
    EVM_CHAIN_ID=8118
    cp /app/config/mainnet-genesis.json $NODE_HOME/config/genesis.json
    ;;
esac

case $NODE_TYPE in
  API)
    OPTIONS="--rpc.laddr tcp://0.0.0.0:26657 \
             --api.enable \
             --api.address tcp://0.0.0.0:1317 \
             --json-rpc.enable \
             --json-rpc.address 0.0.0.0:8545 \
             --json-rpc.ws-address 0.0.0.0:8546 \
             --json-rpc.api eth,txpool,personal,net,debug,web3 \
             --evm.evm-chain-id $EVM_CHAIN_ID \
             --minimum-gas-prices=2048130280389041ashm \
             --pruning nothing"
    ;;
  VALIDATOR)
    OPTIONS='--p2p.laddr tcp://0.0.0.0:27656'
    ;;
  SENTRY)
    # ---
    # Comma separated list of peer IDs to keep private (will not be gossiped to other peers, we've to add this in below sentry start command flags)
    # Example ID: 553bbeeac6c297e228c89046d28a5351c93c3a29
    # ---
    # --p2p.private_peer_ids string
    OPTIONS="--p2p.laddr "tcp://$P2P_ADDRESS:$P2P_PORT" --p2p.private_peer_ids $PRIVATE_PEER_IDS --p2p.unconditional_peer_ids $UNCONDITIONAL_PEER_IDS --rpc.laddr "tcp://$RPC_ADDRESS:$RPC_PORT" --pruning nothing"
    ;;
esac

/app/shardeumd start --home $NODE_HOME --chain-id $CHAIN_ID --p2p.seeds "$PEERS" --p2p.persistent_peers "$PEERS" $OPTIONS
