#!/bin/bash

# Default RPC URL and Paths
DEFAULT_RPC_URL="http://sl-bootnode:8545"
PRIMEV_DIR="$HOME/.primev"
GETH_POA_PATH="$PRIMEV_DIR/go-ethereum"
CONTRACTS_PATH="$PRIMEV_DIR/contracts"
MEV_COMMIT_PATH="$PRIMEV_DIR/mev-commit"
DOCKER_NETWORK_NAME="primev_net"
MEV_COMMIT_BRANCH="main"
GETH_POA_BRANCH="master"
CONTRACTS_BRANCH="main"

# Function to initialize the environment
initialize_environment() {
    create_primev_dir
    create_docker_network
    clone_repos
    update_repos
    checkout_branch
}

# Function to create a Docker network
create_docker_network() {
    echo "Creating Docker network: $DOCKER_NETWORK_NAME..."
    if ! docker network inspect $DOCKER_NETWORK_NAME >/dev/null 2>&1; then
        docker network create --driver bridge --subnet 172.29.0.0/16 $DOCKER_NETWORK_NAME
    else
        echo "Network $DOCKER_NETWORK_NAME already exists."
    fi
}

# Function to create the primev directory
create_primev_dir() {
    echo "Creating directory $PRIMEV_DIR..."
    mkdir -p "$PRIMEV_DIR"
}

# Function to clone all repositories
clone_repos() {
    echo "Cloning repositories under $PRIMEV_DIR..."
    # Clone only if the directory doesn't exist
    [ ! -d "$GETH_POA_PATH" ] && git clone https://github.com:primevprotocol/go-ethereum.git "$GETH_POA_PATH"
    [ ! -d "$CONTRACTS_PATH" ] && git clone https://github.com:primevprotocol/contracts.git "$CONTRACTS_PATH"
    [ ! -d "$MEV_COMMIT_PATH" ] && git clone https://github.com:primevprotocol/mev-commit.git "$MEV_COMMIT_PATH"
}

# Function to checkout a specific branch for all repositories
# If no branch is specified, the default branch is used
checkout_branch() {
    echo "Checking out branch $MEV_COMMIT_BRANCH for mev-commit..."
    git -C "$MEV_COMMIT_PATH" checkout "$MEV_COMMIT_BRANCH"
    echo "Checking out branch $GETH_POA_BRANCH for go-ethereum..."
    git -C "$GETH_POA_PATH" checkout "$GETH_POA_BRANCH"
    echo "Checking out branch $CONTRACTS_BRANCH for contracts..."
    git -C "$CONTRACTS_PATH" checkout "$CONTRACTS_BRANCH"
}

# Function to pull latest changes for all repositories
update_repos() {
    echo "Updating repositories in $PRIMEV_DIR..."
    git -C "$GETH_POA_PATH" pull
    git -C "$CONTRACTS_PATH" pull
    git -C "$MEV_COMMIT_PATH" pull
}


start_settlement_layer() {
    local datadog_key=$1

    git clone https://github.com:primevprotocol/go-ethereum.git "$GETH_POA_PATH"
    echo "Starting Settlement Layer..."

    cat > "$GETH_POA_PATH/geth-poa/.env" <<EOF
CONTRACT_DEPLOYER_PRIVATE_KEY=0xc065f4c9a6dda0785e2224f5af8e473614de1c029acf094f03d5830e2dd5b0ea
NODE1_PRIVATE_KEY=0xe82a054e06f89598485134b4f2ce8a612ce7f7f7e14e650f9f20b30efddd0e57
NODE2_PRIVATE_KEY=0xb17b77fe56797c1a6c236f628d25ede823496af371b3fec858a7a6beff07696b
RELAYER_PRIVATE_KEY=0xa0d74f611ee519f3fd4a84236ee24b955df2a3f40632f404ca46e0b17f696df3
NEXT_PUBLIC_WALLET_CONNECT_ID=
DD_KEY=${datadog_key}
EOF

    export AGENT_BASE_IMAGE=nil
    export L2_NODE_URL=nil

    # Run Docker Compose
    docker compose --profile settlement -f "$GETH_POA_PATH/geth-poa/docker-compose.yml" up -d --build

}

start_mev_commit() {
    local datadog_key=$1
    echo "Starting MEV-Commit..."
    DD_KEY="$datadog_key" docker compose --profile non-datadog -f "$MEV_COMMIT_PATH/integration-compose.yml" up --build -d
}

deploy_contracts() {
    local rpc_url=${1:-$DEFAULT_RPC_URL}
    local chain_id=${2:-17864}  # Default chain ID
    local private_key=${3:-"0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"} # Default private key
    echo "Building Contract Deployment Image..."

    # Path to the Dockerfile in the contracts repository
    DOCKERFILE_PATH="$CONTRACTS_PATH"

    # Ensure the latest contracts repo is being used
    git -C "$DOCKERFILE_PATH" pull

    # Build the Docker image for contract deployment
    docker build -t contract-deployer "$DOCKERFILE_PATH"

    # Wait for the Geth POA network to be up and running
    echo "Waiting for Geth POA network to be fully up..."
    sleep 10

    # Run the Docker container to deploy the contracts
    echo "Deploying Contracts with RPC URL: $rpc_url, Chain ID: $chain_id, and Private Key: [HIDDEN]"
    docker run --rm --network "$DOCKER_NETWORK_NAME" \
        -e RPC_URL="$rpc_url" \
        -e CHAIN_ID="$chain_id" \
        -e PRIVATE_KEY="$private_key" \
        contract-deployer
}


stop_services() {
    service=$1
    echo "Stopping Docker Compose services..."

    case $service in
        "sl")
            docker compose -f "$GETH_POA_PATH/geth-poa/docker-compose.yml" down
            ;;
        "mev-commit")
            docker compose -f "$MEV_COMMIT_PATH/integration-compose.yml" down
            ;;
        "all")
            docker compose -f "$GETH_POA_PATH/geth-poa/docker-compose.yml" down
            docker compose -f "$MEV_COMMIT_PATH/integration-compose.yml" down
            ;;
        *)
            echo "Invalid service: $service"
            echo "Valid services: geth-poa, mev-commit, all"
            return 1
    esac

    echo "Service(s) stopped."
}

cleanup() {
    echo "Cleaning up..."
    make -C "$GETH_POA_PATH" clean-dbs
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

# Main script 
case "$1" in
    sl)
	initialize_environment
        start_settlement_layer "$datadog_key"
        ;;
    deploy_contracts)
	deploy_contracts "$rpc_url"
	;;
    start)
        initialize_environment
        rpc_url=${2:-$DEFAULT_RPC_URL}
        datadog_key=${3:-""}
        start_settlement_layer "$datadog_key"
        deploy_contracts "$rpc_url"
        start_mev_commit "$datadog_key"
        ;;
    stop)
        stop_services "$2"
        ;;
    update)
        create_primev_dir
        update_repos
        ;;
    cleanup)
        cleanup
        ;;
    *)
        echo "Usage: $0 {start|update|cleanup} [rpc-url] [datadog-key]"
        exit 1
esac

exit 0
