#!/bin/sh

set -ex

# Common/default env vars required for all node types
COMMON_ENVS="PEERS SHARDEUM_NETWORK KEYRING_BACKEND GENESIS_URL NODE_HOME NODE_TYPE ALLOW_DUPLICATE_IP"

# Node-type specific env vars
RPC_ENVS="MIN_GAS JSON_RPC_ADDRESS JSON_RPC_PORT JSON_RPC_WS_PORT P2P_ADDRESS P2P_PORT RPC_ADDRESS RPC_PORT GRPC_ADDRESS GRPC_PORT API_ADDRESS API_PORT"
SENTRY_ENVS="P2P_ADDRESS P2P_PORT RPC_ADDRESS RPC_PORT PRIVATE_PEER_IDS"
VALIDATOR_ENVS="P2P_ADDRESS P2P_PORT RPC_ADDRESS RPC_PORT UNCONDITIONAL_PEER_IDS"

# Start with the common ones
REQUIRED_ENVS="$COMMON_ENVS"

# Add node-type-specific vars
case "$NODE_TYPE" in
  RPC)
    REQUIRED_ENVS="$REQUIRED_ENVS $RPC_ENVS"
    ;;
  SENTRY)
    REQUIRED_ENVS="$REQUIRED_ENVS $SENTRY_ENVS"
    ;;
  VALIDATOR)
    REQUIRED_ENVS="$REQUIRED_ENVS $VALIDATOR_ENVS"
    ;;
  *)
    echo "Unknown NODE_TYPE: $NODE_TYPE"
    exit 1
    ;;
esac

# Check for missing env vars
for ENVAR in $REQUIRED_ENVS; do
  eval value=\$$ENVAR
  if [ -z "$value" ]; then
    echo "Missing required env variable $ENVAR"
    missing=1
  fi
done

if [ $missing ]; then exit 1; fi

if [ ! -e "$NODE_HOME/data/state.db" ] && [ ! -e "$NODE_HOME/config/genesis.json" ]; then
  if [ -z "$SHARDEUM_NETWORK" ]; then
    echo "Uninitialized nodes require a SHARDEUM_NETWORK env var and will copy genesis file accordingly"
    exit 1
  fi

  # Get CHAIN_ID on the basis of SHARDEUM_NETWORK used
  # ---
  CONFIG_FILE="/app/config/environments/$SHARDEUM_NETWORK.json"
  # Read network configuration
  CHAIN_ID=$(jq -r '.chain_id' "$CONFIG_FILE")
  echo "############################################################"
  echo "Using CHAIN_ID ===> $CHAIN_ID"
  echo "############################################################"

  # Init
  # ---
  echo "Running Init -------------------------------------"
  /app/shardeumd init docker --home $NODE_HOME --chain-id $CHAIN_ID

  # # Copy genesis file from /app/config/environments/$SHARDEUM_NETWORK-genesis.genesis.json to $NODE_HOME/config/genesis.json
  # # ---
  # echo "Copying genesis from /app/config/environments/$SHARDEUM_NETWORK-genesis.genesis.json =====> $NODE_HOME/config/genesis.json"
  # cp /app/config/environments/$SHARDEUM_NETWORK-genesis.genesis.json $NODE_HOME/config/genesis.json

  # Download genesis file from $GENESIS_URL to $NODE_HOME/config/genesis.json
  # ---
  echo "Downloading genesis from to =====> $NODE_HOME/config/genesis.json"
  wget $GENESIS_URL -O $NODE_HOME/config/genesis.json


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


  if [ "$ALLOW_DUPLICATE_IP" = true ]; then
    # Allow duplicate IP's (only needed to allow_duplicate_ip enabled for local network setup and testing purpose, so commenting it for now)
    # ---
    sed -i "s/allow_duplicate_ip = false/allow_duplicate_ip = true/" "$NODE_HOME/config/config.toml"
  fi

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
    sed -i '/^\[p2p\]/,/^\[/{s/^laddr *=.*/laddr = ""/}' "$NODE_HOME/config/config.toml"
    grep -A5 "\[p2p\]" $NODE_HOME/config/config.toml

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

  if [ "$NODE_TYPE" = "VALIDATOR" ]; then
    ###
    # set rpc.laddr as ""
    # ---
    sed -i '/^\[rpc\]/,/^\[/{s/^laddr *=.*/laddr = ""/}' "$NODE_HOME/config/config.toml"
    grep -A5 "\[rpc\]" $NODE_HOME/config/config.toml

    ###
    # set max_num_inbound_peers to 10
    # ---
    sed -i 's/^max_num_inbound_peers = .*/max_num_inbound_peers = 10/' "$NODE_HOME/config/config.toml"
    cat $NODE_HOME/config/config.toml | grep max_num_inbound_peers

  fi
fi

# case $SHARDEUM_NETWORK in
#   testnet)
#     cp /app/config/testnet-genesis.json $NODE_HOME/config/genesis.json
#     ;;
# esac

# Get CHAIN_ID on the basis of SHARDEUM_NETWORK used
# ---
CONFIG_FILE="/app/config/environments/$SHARDEUM_NETWORK.json"
# Read network configuration
echo "Getting CHAIN_ID"
CHAIN_ID=$(jq -r '.chain_id' "$CONFIG_FILE")
echo "############################################################"
echo "Using CHAIN_ID ===> $CHAIN_ID"
echo "############################################################"

### get node-id
# ---
echo "##############################################################################################################################"
echo "############################################################"
echo "Node's ID ===> $(/app/shardeumd cometbft show-node-id --home $NODE_HOME)"
echo "############################################################"
echo "##############################################################################################################################"

####################################################################################################################################

# Start command parameters for each
case $NODE_TYPE in
  RPC)
    OPTIONS="--p2p.laddr "tcp://$P2P_ADDRESS:$P2P_PORT" --rpc.laddr "tcp://$RPC_ADDRESS:$RPC_PORT" --grpc.address "$GRPC_ADDRESS:$GRPC_PORT" --api.enable --json-rpc.enable --json-rpc.address "$JSON_RPC_ADDRESS:$JSON_RPC_PORT" --json-rpc.ws-address "$JSON_RPC_ADDRESS:$JSON_RPC_WS_PORT" --json-rpc.api eth,txpool,personal,net,debug,web3 --minimum-gas-prices="$MIN_GAS" --pruning nothing"
    ;;

  VALIDATOR)
    # Comma separated list of nodeID’s. These nodes will be connected to no matter the limits of inbound and outbound peers. This is useful for when sentry nodes have full address books
    # ---
    # unconditional_peer_ids = "sentry_ids"  # Always stay connected
    ############
    # NEVER share peers
    # This turns the peer exchange reactor on or off for a node. When pex=false, only the persistent_peers list is available for connection.
    # ---
    # --p2p.pex enable/disable Peer-Exchange (default true)
    OPTIONS="--p2p.laddr "tcp://$P2P_ADDRESS:$P2P_PORT" --rpc.laddr "tcp://$RPC_ADDRESS:$RPC_PORT" --p2p.unconditional_peer_ids $UNCONDITIONAL_PEER_IDS --p2p.pex false --pruning nothing"
    ;;

  SENTRY)
    # Comma separated list of peer IDs to keep private (will not be gossiped to other peers, we've to add this in below sentry start command flags)
    # Example ID: 553bbeeac6c297e228c89046d28a5351c93c3a29
    # ---
    # --p2p.private_peer_ids string
    OPTIONS="--p2p.laddr "tcp://$P2P_ADDRESS:$P2P_PORT" --p2p.private_peer_ids $PRIVATE_PEER_IDS --rpc.laddr "tcp://$RPC_ADDRESS:$RPC_PORT" --pruning nothing"
    ;;
esac

# /app/shardeumd start --home $NODE_HOME --chain-id shardeum-testnet --p2p.seeds "$PEERS" --p2p.persistent_peers "$PEERS" $OPTIONS
/app/shardeumd start --home $NODE_HOME --chain-id $CHAIN_ID --p2p.seeds "$PEERS" --p2p.persistent_peers "$PEERS" $OPTIONS

################################################################################################################################################################

# if [ "$NODE_TYPE" = "RPC" ]; then
#   /app/shardeumd start \
#   --home $NODE_HOME \
#   --chain-id $CHAIN_ID \
#   --p2p.seeds "$PEERS" \
#   --p2p.persistent_peers "$PEERS" \
#   --p2p.laddr "tcp://$P2P_ADDRESS:$P2P_PORT" \
#   --rpc.laddr "tcp://$RPC_ADDRESS:$RPC_PORT" \
#   --grpc.address "$GRPC_ADDRESS:$GRPC_PORT" \
#   --api.enable \
#   --json-rpc.enable \
#   --json-rpc.address "$JSON_RPC_ADDRESS:$JSON_RPC_PORT" \
#   --json-rpc.ws-address "$JSON_RPC_ADDRESS:$JSON_RPC_WS_PORT" \
#   --json-rpc.api eth,txpool,personal,net,debug,web3 \
#   --minimum-gas-prices="$MIN_GAS" \
#   --pruning nothing
# fi

# if [ "$NODE_TYPE" = "VALIDATOR" ]; then
#   /app/shardeumd start \
#   --home $NODE_HOME \
#   --chain-id $CHAIN_ID \
#   --p2p.seeds "$PEERS" \
#   --p2p.persistent_peers "$PEERS" \
#   --p2p.laddr "tcp://$P2P_ADDRESS:$P2P_PORT" \
#   --rpc.laddr "tcp://$RPC_ADDRESS:$RPC_PORT" \
#   --pruning nothing
# fi

# # Comma separated list of peer IDs to keep private (will not be gossiped to other peers, we've to add this in below sentry start command flags)
# # Example ID: 553bbeeac6c297e228c89046d28a5351c93c3a29
# # ---
# # --p2p.private_peer_ids string
# if [ "$NODE_TYPE" = "SENTRY" ]; then
#   /app/shardeumd start \
#   --home $NODE_HOME \
#   --chain-id $CHAIN_ID \
#   --p2p.seeds "$PEERS" \
#   --p2p.persistent_peers "$PEERS" \
#   --p2p.laddr "tcp://$P2P_ADDRESS:$P2P_PORT" \
#   --pruning nothing
# fi
