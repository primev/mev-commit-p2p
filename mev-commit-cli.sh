#!/bin/bash

# Default RPC URL and Paths
L1_RPC_BASE_URL=https://sepolia.infura.io/v3
DEFAULT_RPC_URL="http://sl-bootnode:8545"
PRIMEV_DIR="$HOME/.primev"

GETH_REPO_NAME="mev-commit-geth"
CONTRACT_REPO_NAME="contracts"
MEV_COMMIT_REPO_NAME="mev-commit"
ORACLE_REPO_NAME="mev-commit-oracle"

GETH_POA_PATH="$PRIMEV_DIR/$GETH_REPO_NAME"
CONTRACTS_PATH="$PRIMEV_DIR/$CONTRACT_REPO_NAME"
MEV_COMMIT_PATH="$PRIMEV_DIR/$MEV_COMMIT_REPO_NAME"
ORACLE_PATH="$PRIMEV_DIR/$ORACLE_REPO_NAME"

DOCKER_NETWORK_NAME="primev_net"
MEV_COMMIT_BRANCH="main"
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
    [ ! -d "$GETH_POA_PATH" ] && git clone https://github.com/primevprotocol/$GETH_REPO_NAME.git "$GETH_POA_PATH"
    [ ! -d "$CONTRACTS_PATH" ] && git clone https://github.com/primevprotocol/$CONTRACT_REPO_NAME.git "$CONTRACTS_PATH"
    [ ! -d "$MEV_COMMIT_PATH" ] && git clone https://github.com/primevprotocol/$MEV_COMMIT_REPO_NAME.git "$MEV_COMMIT_PATH"
    [ ! -d "$ORACLE_PATH" ] && git clone https://github.com/primevprotocol/$ORACLE_REPO_NAME.git "$ORACLE_PATH"
}

# Function to checkout a specific branch for all repositories
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
    HYPERLANE_DEPLOYER_PRIVATE_KEY=0xc065f4c9a6dda0785e2224f5af8e473614de1c029acf094f03d5830e2dd5b0ea
    NODE1_PRIVATE_KEY=0xe82a054e06f89598485134b4f2ce8a612ce7f7f7e14e650f9f20b30efddd0e57
    NODE2_PRIVATE_KEY=0xb17b77fe56797c1a6c236f628d25ede823496af371b3fec858a7a6beff07696b
    RELAYER_PRIVATE_KEY=0xa0d74f611ee519f3fd4a84236ee24b955df2a3f40632f404ca46e0b17f696df3
    NEXT_PUBLIC_WALLET_CONNECT_ID=0x074ac60cba235536b25b262f66dee686
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
    BIDDER_REGISTRY=0x390066a15e1048445F1B1b69Ba98AC4cb5e91c52
    PROVIDER_REGISTRY=0xeA73E67c2E34C4E02A2f3c5D416F59B76e7617fC
    PRECONF_CONTRACT=0xBB632720f817792578060F176694D8f7230229d9
    RPC_URL=${rpc_url}
    PRIVATE_KEY=ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
    L1_RPC_URL="${L1_RPC_BASE_URL}/${sepolia_key}"
EOF


    # Check if datadog_key is empty
    if [ -z "$datadog_key" ]; then
        echo "DD_KEY is empty, so no agents will be started."
        # Run Docker Compose without --profile agent
        docker compose --profile e2etest -f "$MEV_COMMIT_PATH/e2e-compose.yml" up --build -d
    else
        # Run Docker Compose with --profile agent
        DD_KEY="$datadog_key" docker compose --profile e2etest --profile agent -f "$MEV_COMMIT_PATH/e2e-compose.yml" up --build -d
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

    # Ensure the latest contracts repo is being used
    git -C "$CONTRACTS_PATH" pull

    # Build the Docker image for contract deployment
    docker build -t contract-deployer "$CONTRACTS_PATH"

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
        && /deploy_create2.sh ${rpc_url}"

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
    local datadog_key=$2
    cat > "$ORACLE_PATH/.env" <<EOF
    L1_URL="${L1_RPC_BASE_URL}/${sepolia_key}"
    INTEGREATION_TEST=true
    DB_HOST=localhost
    POSTGRES_PASSWORD=oracle_pass
    DD_KEY=${datadog_key}
EOF
    # Run Docker Compose
    DEPLOY_ENV=e2e DD_KEY="$datadog_key" docker compose -f "$ORACLE_PATH/docker-compose.yml" up -d --build
}

stop_oracle(){
    # Run Docker Compose
    DEPLOY_ENV=e2e DD_KEY=nil docker compose -f "$ORACLE_PATH/docker-compose.yml" down
}

start_bridge(){
    local public_rpc_url=${1:-$DEFAULT_RPC_URL}
    local rpc_url=${2:-$DEFAULT_RPC_URL}
    AGENT_BASE_IMAGE=gcr.io/abacus-labs-dev/hyperlane-agent@sha256:854f92966eac6b49e5132e152cc58168ecdddc76c2d390e657b81bdaf1396af0 PUBLIC_SETTLEMENT_RPC_URL="$public_rpc_url" SETTLEMENT_RPC_URL="$rpc_url" docker compose -f "$GETH_POA_PATH/geth-poa/docker-compose.yml" --profile bridge up -d --build
}

stop_bridge(){
    AGENT_BASE_IMAGE=gcr.io/abacus-labs-dev/hyperlane-agent@sha256:854f92966eac6b49e5132e152cc58168ecdddc76c2d390e657b81bdaf1396af0 PUBLIC_SETTLEMENT_RPC_URL="$public_rpc_url" SETTLEMENT_RPC_URL="$rpc_url" docker compose -f "$GETH_POA_PATH/geth-poa/docker-compose.yml" --profile bridge down
}

clean() {
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

stop_services() {
    service=$1
    echo "Stopping Docker Compose services..."

    case $service in
        "sl")
            docker compose -f "$GETH_POA_PATH/geth-poa/docker-compose.yml" down
            ;;
        "oracle")
            stop_oracle  # Assuming stop_oracle is a function you've defined elsewhere
            ;;
        "bridge")
            stop_bridge
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
            echo "Valid services: sl, oracle, mev-commit, all"
            return 1
    esac

    echo "Service(s) stopped."
}


start_service() {
    local service_name=$1
    case $service_name in
        "all")
            start_settlement_layer "$datadog_key"
            deploy_contracts "$rpc_url"
            start_mev_commit "$datadog_key"
            start_oracle "$sepolia_key" "$datadog_key"
            start_bridge
            ;;
        "e2e")
            initialize_environment
            start_settlement_layer "$datadog_key"
            deploy_contracts "$rpc_url"
            start_mev_commit_e2e "--sepolia-key=$sepolia_key" "--datadog-key=$datadog_key"
            sleep 12
            start_oracle "$sepolia_key" "$datadog_key"
            start_bridge "$public_rpc_url"
            ;;
        "mev-commit")
            start_mev_commit "$datadog_key"
            ;;
        "oracle")
            start_oracle "$sepolia_key" "$datadog_key"
            ;;
        "sl")
            start_settlement_layer "$datadog_key"
            ;;
        "bridge")
            start_bridge "$public_rpc_url"
            ;;
        "minimal")
            initialize_environment
            start_settlement_layer "$datadog_key"
            deploy_contracts "$rpc_url"
            start_mev_commit_minimal
            ;;
        *)
            echo "Invalid service name: $service_name"
            echo "Valid services: all, e2e, oracle, sl, bridge"
            return 1
            ;;
    esac
}

# Function to display help
show_help() {
    echo "Usage: $0 [command] [service(s)] [options]"
    echo ""
    echo "Commands:"
    echo "  deploy_contracts       Deploy contracts"
    echo "  start [services]       Start specified services. Available services: all, e2e, mev-commit, oracle, sl, bridge, minimal"
    echo "  stop [service]         Stop specified service. Available services: sl, mev-commit, all"
    echo "  update                 Update repositories"
    echo "  clean                  Cleanup Docker"
    echo ""
    echo "Options:"
    echo "  -h, --help             Show this help message"
    echo "  --rpc-url URL          Set the internal RPC URL for mev-commit-geth"
    echo "  --public-rpc-url URL   Set the public RPC URL for mev-commit-geth"
    echo "  --datadog-key KEY      Set the Datadog key"
    echo "  --sepolia-key KEY      Set the Sepolia key"
    echo ""
    echo "Examples:"
    echo "  $0 start all --rpc-url http://localhost:8545  Start all services with a specific RPC URL"
    echo "  $0 start e2e --datadog-key abc123             Start only the e2e service with a Datadog key"
    echo "  $0 start oracle                               Start only the oracle service"
    echo "  $0 start sl                                   Start only the settlement layer service"
    echo "  $0 stop sl                                    Stop the settlement layer service"
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
        --public-rpc-url)
            public_rpc_url="$2"
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
        start|stop|deploy_contracts|update|clean)
            command="$1"
            shift
            # If additional arguments are present after the command, they are captured as service names or additional options
            service_names=()
            while [[ "$#" -gt 0 ]] && [[ "$1" != "--"* ]]; do
                service_names+=("$1")
                shift
            done
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
    start)
        if [ ${#service_names[@]} -eq 0 ]; then
            echo "No service specified. Starting all services."
            start_service "all"
        else
            for service_name in "${service_names[@]}"; do
                start_service "$service_name"
            done
        fi
        ;;
    deploy_contracts)
        deploy_contracts "$rpc_url"
        ;;
    stop)
        if [ -z "${service_names[0]}" ]; then
            echo "No service specified for stopping."
            exit 1
        else
            stop_services "${service_names[0]}"
        fi
        ;;
    update)
        update_repos
        ;;
    clean)
        clean
        ;;
esac

exit 0