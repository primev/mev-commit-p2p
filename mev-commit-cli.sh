#!/bin/bash

# Default RPC URL and Paths
DEFAULT_RPC_URL="http://sl-bootnode:8545"
PRIMEV_DIR="$HOME/.primev"
GETH_POA_PATH="$PRIMEV_DIR/mev-commit-geth"
CONTRACTS_PATH="$PRIMEV_DIR/contracts"
MEV_COMMIT_PATH="$PRIMEV_DIR/mev-commit"
ORACLE_PATH="$PRIMEV_DIR/mev-oracle"
DOCKER_NETWORK_NAME="primev_net"
MEV_COMMIT_BRANCH="ckartik/oracle-testing"
GETH_POA_BRANCH="master"
CONTRACTS_BRANCH="main"
ORACLE_BRANCH="main"

# Default values for optional arguments
rpc_url=$DEFAULT_RPC_URL
datadog_key=""
command=""

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
    [ ! -d "$GETH_POA_PATH" ] && git clone https://github.com/primevprotocol/go-ethereum.git "$GETH_POA_PATH"
    [ ! -d "$CONTRACTS_PATH" ] && git clone https://github.com/primevprotocol/contracts.git "$CONTRACTS_PATH"
    [ ! -d "$MEV_COMMIT_PATH" ] && git clone https://github.com/primevprotocol/mev-commit.git "$MEV_COMMIT_PATH"
    [ ! -d "$ORACLE_PATH" ] && git clone https://github.com/primevprotocol/mev-oracle.git "$ORACLE_PATH"
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
    echo "Checking out branch $ORACLE_BRANCH for oracle..."
    git -C "$ORACLE_PATH" checkout "$ORACLE_BRANCH"
}

# Function to pull latest changes for all repositories
update_repos() {
    echo "Updating repositories in $PRIMEV_DIR..."
    git -C "$GETH_POA_PATH" pull
    git -C "$CONTRACTS_PATH" pull
    git -C "$MEV_COMMIT_PATH" pull
    git -C "$ORACLE_PATH" pull
}

start_settlement_layer() {
    local datadog_key=$1

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

start_mev_commit_minimal() {
    echo "Starting MEV-Commit..."
    docker compose --profile minimal-setup -f "$MEV_COMMIT_PATH/integration-compose.yml" up --build -d
}


start_mev_commit_e2e() {
    local datadog_key=""
    local sepolia_key=""
    echo "Starting MEV-Commit..."

    # Loop through arguments and process them
    for arg in "$@"
    do
        case $arg in
            --datadog-key=*)
            datadog_key="${arg#*=}"
            shift # Remove --datadog-key= from processing
            ;;
            --sepolia-key=*)
            sepolia_key="${arg#*=}"
            shift # Remove --sepolia-key= from processing
            ;;
            *)
            # Unknown option
            ;;
        esac
    done
    echo "Setting .env file ..."

        # Create or overwrite the .env file
    cat > "$MEV_COMMIT_PATH/integrationtest/.env" <<EOF
BIDDER_REGISTRY=0xe38B5a8C41f307646F395030992Aa008978E2699
PROVIDER_REGISTRY=0x7fA45D14358B698Bd85a0a2B03720A6Fe4b566d7
PRECONF_CONTRACT=0x8B0F623dCD54cA50CD154B3dDCbB8436E876b019
RPC_URL=http://sl-bootnode:8545
PRIVATE_KEY=ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
L1_RPC_URL=https://sepolia.infura.io/v3/${sepolia_key}
EOF


    # Check if datadog_key is empty
    if [ -z "$datadog_key" ]; then
        echo "DD_KEY is empty, so no agents will be started."
        # Run Docker Compose without --profile agent
        docker compose --profile main -f "$MEV_COMMIT_PATH/e2e-compose.yml" up --build -d
    else
        # Run Docker Compose with --profile agent
        DD_KEY="$datadog_key" docker compose --profile main --profile agent -f "$MEV_COMMIT_PATH/e2e-compose.yml" up --build -d
    fi
}

start_mev_commit() {
    local datadog_key=$1

    echo "Starting MEV-Commit..."

    # Check if datadog_key is empty
    if [ -z "$datadog_key" ]; then
        echo "DD_KEY is empty, so no agents will be started."
        # Run Docker Compose without --profile agent
        docker compose --profile integration-test -f "$MEV_COMMIT_PATH/integration-compose.yml" up --build -d
    else
        # Run Docker Compose with --profile agent
        DD_KEY="$datadog_key" docker compose --profile integration-test --profile agent -f "$MEV_COMMIT_PATH/integration-compose.yml" up --build -d
    fi
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

    # Deploy create2 proxy from alpine container
    chmod +x "$GETH_POA_PATH/geth-poa/util/deploy_create2.sh"
    docker run \
        --rm \
        --network "$DOCKER_NETWORK_NAME" \
        -v "$GETH_POA_PATH/geth-poa/util/deploy_create2.sh:/deploy_create2.sh" \
        alpine /bin/sh -c \
        "apk add --no-cache curl jq \
        && /deploy_create2.sh http://sl-bootnode:8545"

    # Run the Docker container to deploy the contracts
    echo "Deploying Contracts with RPC URL: $rpc_url, Chain ID: $chain_id, and Private Key: [HIDDEN]"
    docker run --rm --network "$DOCKER_NETWORK_NAME" \
        -e RPC_URL="$rpc_url" \
        -e CHAIN_ID="$chain_id" \
        -e PRIVATE_KEY="$private_key" \
        contract-deployer
}

start_oracle(){
    local sepolia_key=$1
    local starting_block_number=$2
    local datadog_key=$3
    cat > "$ORACLE_PATH/.env" <<EOF
L1_URL=https://sepolia.infura.io/v3/${sepolia_key}
STARTING_BLOCK=${starting_block_number}
INTEGREATION_TEST=true
DB_HOST=localhost
POSTGRES_PASSWORD=oracle_pass
DD_KEY=${datadog_key}
EOF

    # Run Docker Compose
    DEPLOY_ENV=e2e DD_KEY="$datadog_key" docker compose -f "$ORACLE_PATH/docker-compose.yml" up -d --build

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

# Function to display help
show_help() {
    echo "Usage: $0 [options] [command]"
    echo ""
    echo "Commands:"
    echo "  sl                     Start the settlement layer"
    echo "  deploy_contracts       Deploy contracts"
    echo "  start                  Start the environment"
    echo "  start-minimal          Start the minimal environment"
    echo "  start-e2e              Start the minimal environment for oracle with live bids from Infura"
    echo "  stop                   Stop services"
    echo "  update                 Update repositories"
    echo "  cleanup                Cleanup Docker"
    echo ""
    echo "Options:"
    echo "  -h, --help             Show this help message"
    echo "  --rpc-url URL          Set the RPC URL"
    echo "  --datadog-key KEY      Set the Datadog key"
    echo "  --sepolia-key KEY      Set the Sepolia key"
    echo "  --starting-block-number NUMBER      Set the starting block number for oracle"
    echo ""
}

# Parse command line options
while [[ "$#" -gt 0 ]]; do
    case "$1" in
        -h|--help)
            show_help
            exit 0
            ;;
        --rpc-url)
            rpc_url="$2"
            shift 2
            ;;
        --datadog-key)
            datadog_key="$2"
            shift 2
            ;;
        --sepolia-key)
            sepolia_key="$2"
            shift 2
            ;;
        --starting-block-number)
            starting_block_number="$2"
            shift 2
            ;;
        sl|deploy_contracts|start|start-minimal|start-e2e|stop|update|cleanup)
            if [[ -z "$command" ]]; then
                command="$1"
            else
                echo "Multiple commands specified. Please use only one command."
                exit 1
            fi
            shift
            ;;
        *)
            echo "Invalid option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Check if a command has been specified
if [[ -z "$command" ]]; then
    echo "No command specified."
    show_help
    exit 1
fi

# Main script logic based on the command variable
case "$command" in
    sl)
        initialize_environment
        start_settlement_layer "$datadog_key"
        ;;
    deploy_contracts)
        deploy_contracts "$rpc_url"
        ;;
    start)
        initialize_environment
        start_settlement_layer "$datadog_key"
        deploy_contracts "$rpc_url"
        start_mev_commit "$datadog_key"
        ;;
    start-e2e)
        initialize_environment
        start_settlement_layer "$datadog_key"
        deploy_contracts "$rpc_url"
        start_mev_commit_e2e "--sepolia-key=$sepolia_key" "--datadog-key=$datadog_key"
        sleep 12
        start_oracle "$sepolia_key" "$starting_block_number" "$datadog_key"
        ;;
    start-minimal)
        initialize_environment
        start_settlement_layer "$datadog_key"
        deploy_contracts "$rpc_url"
        start_mev_commit_minimal
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
        echo "Invalid command: $command"
        show_help
        exit 1
        ;;
esac

exit 0

