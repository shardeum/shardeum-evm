#!/bin/sh

set -ex

for ENVAR in PEERS SHARDEUM_NETWORK GENESIS_SOURCE CHAIN_ID KEYRING_BACKEND NODE_HOME NODE_TYPE MIN_GAS JSON_RPC_ADDRESS JSON_RPC_PORT JSON_RPC_WS_PORT P2P_ADDRESS P2P_PORT RPC_ADDRESS RPC_PORT GRPC_ADDRESS GRPC_PORT API_ADDRESS API_PORT; do
  eval value=\$$ENVAR
  if [ -z "$value" ]; then
    echo "Missing required env variable $ENVAR"
    missing=1
  fi
done

if [ $missing ]; then exit 1; fi

if [ ! -e "$NODE_HOME/data/state.db" ] && [ ! -e "$NODE_HOME/config/genesis.json" ]; then
  if [ -z "$GENESIS_SOURCE" ]; then
    echo "Uninitialized nodes require a GENESIS_SOURCE env var"
    exit 1
  fi
  /app/shardeumd init docker --home $NODE_HOME --chain-id $CHAIN_ID
  # curl -s http://$GENESIS_SOURCE/genesis | jq -r '.result.genesis' > "$NODE_HOME/config/genesis.json"
  echo "Downloading genesis json file from $GENESIS_SOURCE"
  curl -s $GENESIS_SOURCE | jq -r '.result.genesis' > "$NODE_HOME/config/genesis.json"
  sleep 5
  echo "Downloaded genesis json file from $GENESIS_SOURCE"
  
  # Set up client configuration (do we need this ???)
cat > "$NODE_HOME/config/client.toml" << EOF
chain-id = "$CHAIN_ID"
keyring-backend = "$KEYRING_BACKEND"
node = "tcp://$RPC_ADDRESS:$RPC_PORT"
broadcast-mode = "sync"
EOF

  cat $NODE_HOME/config/client.toml | grep chain-id
  cat $NODE_HOME/config/client.toml | grep keyring
  cat $NODE_HOME/config/client.toml | grep node
  cat $NODE_HOME/config/client.toml | grep sync

  # to enable the api server in $NODE_HOME/config/app.toml (enabling from the start command if $NODE_TYPE is set to "full-node")
  # ---
  # sed -i '/\[api\]/,+3 s/enable = false/enable = true/' $NODE_HOME/config/app.toml

  # # Allow duplicate IP's (only needed to allow_duplicate_ip enabled for local network setup and testing purpose, so commenting it for now)
  # ---
  # sed -i "s/allow_duplicate_ip = false/allow_duplicate_ip = true/" "$NODE_HOME/config/config.toml"

####################################################################################################################################

  if [ "$NODE_TYPE" = "RPC" ]; then
    ###
    # change max_open_connections (only change the 1st occurance as that's of [rpc])
    # ---
    sed -i '1,/^max_open_connections[[:space:]]*=/{s/^max_open_connections[[:space:]]*=.*/max_open_connections = 300/;}' $NODE_HOME/config/config.toml
    cat $NODE_HOME/config/config.toml | grep max_open_connections

    # change max_subscription_clients
    # ---
    sed -i 's/^max_subscription_clients = .*/max_subscription_clients = 50/' "$NODE_HOME/config/config.toml"
    cat $NODE_HOME/config/config.toml | grep max_subscription_clients

    # change max_subscriptions_per_client
    # ---
    sed -i 's/^max_subscriptions_per_client = .*/max_subscriptions_per_client = 3/' "$NODE_HOME/config/config.toml"
    cat $NODE_HOME/config/config.toml | grep max_subscriptions_per_client

    # change experimental_close_on_slow_client
    # ---
    sed -i 's/^experimental_close_on_slow_client = .*/experimental_close_on_slow_client = true/' "$NODE_HOME/config/config.toml"
    cat $NODE_HOME/config/config.toml | grep experimental_close_on_slow_client

    # set p2p.laddr as ""
    # ---
    # sed -i '/^\[p2p\]/,/^\[/{s/^laddr *=.*/laddr = ""/}' "$NODE_HOME/config/config.toml"
    # cat $NODE_HOME/config/config.toml | grep laddr
  
  fi

  if [ "$NODE_TYPE" = "VALIDATOR" ]; then
    ###
    # set rpc.laddr as ""
    # ---
    sed -i '/^\[rpc\]/,/^\[/{s/^laddr *=.*/laddr = ""/}' "$NODE_HOME/config/config.toml"
    grep -A2 "^\[rpc\]" $NODE_HOME/config/config.toml

  fi
fi

case $SHARDEUM_NETWORK in
  testnet)
    cp /app/config/testnet-genesis.json $NODE_HOME/config/genesis.json
    ;;
esac

####################################################################################################################################

# EMPTY=""

# # Start command parameters for each
# case $NODE_TYPE in
#   RPC)
#     # OPTIONS='--rpc.laddr "tcp://0.0.0.0:26657" --json-rpc.enable --json-rpc.address "0.0.0.0:8545" --api.enable --api.address "tcp://0.0.0.0:1317"'
#     OPTIONS="--p2p.laddr "$EMPTY" --rpc.laddr "tcp://$RPC_ADDRESS:$RPC_PORT" --grpc.address "$GRPC_ADDRESS:$GRPC_PORT" --api.enable --json-rpc.enable --json-rpc.address "$JSON_RPC_ADDRESS:$JSON_RPC_PORT" --json-rpc.ws-address "$JSON_RPC_ADDRESS:$JSON_RPC_WS_PORT" --json-rpc.api eth,txpool,personal,net,debug,web3 --minimum-gas-prices="$MIN_GAS" --pruning nothing"
#     ;;
#   VALIDATOR)
#     OPTIONS="--p2p.laddr "tcp://$P2P_ADDRESS:$P2P_PORT" --rpc.laddr "" --pruning nothing"
#     ;;
#   SENTRY)
#     OPTIONS="--p2p.laddr "tcp://$P2P_ADDRESS:$P2P_PORT" --pruning nothing"
#     ;;
# esac

# # /app/shardeumd start --home $NODE_HOME --chain-id shardeum-testnet --p2p.seeds "$PEERS" --p2p.persistent_peers "$PEERS" $OPTIONS
# /app/shardeumd start --home $NODE_HOME --chain-id $CHAIN_ID --p2p.seeds "$PEERS" --p2p.persistent_peers "$PEERS" $OPTIONS


if [ "$NODE_TYPE" = "RPC" ]; then
  /app/shardeumd start \
  --home $NODE_HOME \
  --chain-id $CHAIN_ID \
  --p2p.seeds "$PEERS" \
  --p2p.persistent_peers "$PEERS" \
  --p2p.laddr "tcp://$P2P_ADDRESS:$P2P_PORT" \
  --rpc.laddr "tcp://$RPC_ADDRESS:$RPC_PORT" \
  --grpc.address "$GRPC_ADDRESS:$GRPC_PORT" \
  --api.enable \
  --json-rpc.enable \
  --json-rpc.address "$JSON_RPC_ADDRESS:$JSON_RPC_PORT" \
  --json-rpc.ws-address "$JSON_RPC_ADDRESS:$JSON_RPC_WS_PORT" \
  --json-rpc.api eth,txpool,personal,net,debug,web3 \
  --minimum-gas-prices="$MIN_GAS" \
  --pruning nothing
fi

if [ "$NODE_TYPE" = "VALIDATOR" ]; then
  /app/shardeumd start \
  --home $NODE_HOME \
  --chain-id $CHAIN_ID \
  --p2p.seeds "$PEERS" \
  --p2p.persistent_peers "$PEERS" \
  --p2p.laddr "tcp://$P2P_ADDRESS:$P2P_PORT" \
  --rpc.laddr "tcp://$RPC_ADDRESS:$RPC_PORT" \
  --pruning nothing
fi

if [ "$NODE_TYPE" = "RPC" ]; then
  /app/shardeumd start \
  --home $NODE_HOME \
  --chain-id $CHAIN_ID \
  --p2p.seeds "$PEERS" \
  --p2p.persistent_peers "$PEERS" \
  --p2p.laddr "tcp://$P2P_ADDRESS:$P2P_PORT" \
  --pruning nothing
fi