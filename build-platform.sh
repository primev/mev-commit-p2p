#!/bin/bash

# Default RPC URL
DEFAULT_RPC_URL="http://localhost:8545"

# Function to start the settlement layer
start_settlement_layer() {
    cd ~/
    git clone git@github.com:primevprotocol/go-ethereum.git
    echo "Starting Settlement Layer..."
    cd go-ethereum/geth-poa
    make up-prod-settlement
    cd - # Return to the original directory
}

# Function to deploy contracts
deploy_contracts() {
    local rpc_url=$1
    echo "Deploying Contracts with RPC URL: $rpc_url..."
    cd ~/
    git clone git@github.com:primevprotocol/rollup-preconf.git
    cd rollup-preconf
    forge script scripts/DeployScripts.s.sol:DeployScript --rpc-url "$rpc_url" --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 --broadcast --chain-id 17864 -vvvv
    cd - # Return to the original directory
}

# Function to start mev-commit
start_mev_commit() {
    local datadog_key=$1
    echo "Starting MEV-Commit..."
    cd - # Return to the original directory
    DD_KEY= docker compose -f integration-compose.yml up --build -d
}

# Function to clean up
cleanup() {
    echo "Cleaning up..."
    cd ~/
    cd go-ethereum/geth-poa
    make clean-dbs
    
    # Docker cleanup script
    echo "Stopping all Docker containers..."
    docker stop $(docker ps -aq)

    echo "Removing all Docker containers..."
    docker rm $(docker ps -aq)

    echo "Removing all Docker images..."
    docker rmi $(docker images -q)

    echo "Removing all Docker volumes..."
    docker volume rm $(docker volume ls -q)

    echo "Removing all Docker networks..."
    docker network ls | grep "bridge\|none\|host" -v | awk '{if(NR>1)print $1}' | xargs -r docker network rm

    echo "Pruning Docker system..."
    docker system prune -a -f --volumes

    echo "Docker cleanup complete."
}


# Main script logic
case "$1" in
    start)
        rpc_url=${2:-$DEFAULT_RPC_URL}  # Use the provided RPC URL or default if not provided
        datadog_key=${3:-""}  # Use the provided Datadog key or empty if not provided
        start_settlement_layer
        deploy_contracts "$rpc_url"

        # Pause for user to update config files
        echo "Please update /integration-test/config/{nodetype}.yml files as needed."
        echo "Press Enter to continue after you have made the updates..."
        echo "Note: If you're on mac, set RPC endpoint to host.internal.docker to connect to localhost"
        read -p " " # This will pause and wait for the user to press Enter

        start_mev_commit "$datadog_key"
        ;;
    cleanup)
        cleanup
        ;;
    *)
        echo "Usage: $0 {start|cleanup} [rpc-url] [datadog-key]"
        exit 1
esac

exit 0