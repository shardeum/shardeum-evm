# Broadcast EIP-712 Tool

A command-line tool to parse and display EIP-712 signed transactions from MetaMask.

## Installation

### I always set these env vars, but not sure how many are actually needed for this flow:
export SHARDEUM_NETWORK=local
export SHARDEUM_CHAIN_ID=shardeum_8117-1
export BINARY="$PWD/build/shardeumd"
export SHARDEUM_CONFIG_DIR="$PWD/config"
echo "setting env vars"
echo "Binary: " $BINARY



### To build the tools run : 

make build-eip712 


### local host the project files:

python -m http.server 8000

### visit the page:
http://localhost:8000/eip712demo/metamask-eip712-delegate.html

- Connect your metamask account  (needs to be one with enough funds for gas/stake)

### three things on the form need updating:  
- acount number 
- sequence 
- the validator operator key to stake.  I ususally get this from the local run dashboard

The account number and sequence will generally be the same between tests but the validator key will change for each network relaunch.
If you send more than one message the go through you have to udpate the sequence number.  (query from chain option is broken, I just use command line)

example: 
ACCOUNT_NUMBER=$(curl -s http://localhost:1317/cosmos/auth/v1beta1/accounts/shardeum1gz02hcrkklw2fe5vv34vh5pzact2dwg06hl3ug | jq -r '.account.account_number')
SEQUENCE=$(curl -s http://localhost:1317/cosmos/auth/v1beta1/accounts/shardeum1gz02hcrkklw2fe5vv34vh5pzact2dwg06hl3ug | jq -r '.account.sequence')
echo "Account Number: $ACCOUNT_NUMBER"
echo "Sequence: $SEQUENCE"


- click "sign with metamask"

- copy the output text and save to a json file 


- run the broadcast tool:

Example:
./build/broadcast-eip712 eip712demo/signed-tx1.json --broadcast






