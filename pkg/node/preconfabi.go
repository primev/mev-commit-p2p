package node

var preConfJson = `
[
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "_providerRegistry",
        "type": "address"
      },
      {
        "internalType": "address",
        "name": "_userRegistry",
        "type": "address"
      },
      {
        "internalType": "address",
        "name": "_oracle",
        "type": "address"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "constructor"
  },
  {
    "inputs": [],
    "name": "ECDSAInvalidSignature",
    "type": "error"
  },
  {
    "inputs": [
      {
        "internalType": "uint256",
        "name": "length",
        "type": "uint256"
      }
    ],
    "name": "ECDSAInvalidSignatureLength",
    "type": "error"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "s",
        "type": "bytes32"
      }
    ],
    "name": "ECDSAInvalidSignatureS",
    "type": "error"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "owner",
        "type": "address"
      }
    ],
    "name": "OwnableInvalidOwner",
    "type": "error"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "account",
        "type": "address"
      }
    ],
    "name": "OwnableUnauthorizedAccount",
    "type": "error"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "address",
        "name": "previousOwner",
        "type": "address"
      },
      {
        "indexed": true,
        "internalType": "address",
        "name": "newOwner",
        "type": "address"
      }
    ],
    "name": "OwnershipTransferred",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "address",
        "name": "signer",
        "type": "address"
      },
      {
        "indexed": false,
        "internalType": "string",
        "name": "txnHash",
        "type": "string"
      },
      {
        "indexed": true,
        "internalType": "uint64",
        "name": "bid",
        "type": "uint64"
      },
      {
        "indexed": false,
        "internalType": "uint64",
        "name": "blockNumber",
        "type": "uint64"
      }
    ],
    "name": "SignatureVerified",
    "type": "event"
  },
  {
    "stateMutability": "payable",
    "type": "fallback"
  },
  {
    "inputs": [],
    "name": "DOMAIN_SEPARATOR_BID",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "DOMAIN_SEPARATOR_PRECONF",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "EIP712_COMMITMENT_TYPEHASH",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "EIP712_MESSAGE_TYPEHASH",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes",
        "name": "_bytes",
        "type": "bytes"
      }
    ],
    "name": "_bytesToHexString",
    "outputs": [
      {
        "internalType": "string",
        "name": "",
        "type": "string"
      }
    ],
    "stateMutability": "pure",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      },
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "name": "blockCommitments",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "commitmentCount",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "name": "commitments",
    "outputs": [
      {
        "internalType": "bool",
        "name": "commitmentUsed",
        "type": "bool"
      },
      {
        "internalType": "address",
        "name": "bidder",
        "type": "address"
      },
      {
        "internalType": "address",
        "name": "commiter",
        "type": "address"
      },
      {
        "internalType": "uint64",
        "name": "bid",
        "type": "uint64"
      },
      {
        "internalType": "uint64",
        "name": "blockNumber",
        "type": "uint64"
      },
      {
        "internalType": "bytes32",
        "name": "bidHash",
        "type": "bytes32"
      },
      {
        "internalType": "string",
        "name": "txnHash",
        "type": "string"
      },
      {
        "internalType": "string",
        "name": "commitmentHash",
        "type": "string"
      },
      {
        "internalType": "bytes",
        "name": "bidSignature",
        "type": "bytes"
      },
      {
        "internalType": "bytes",
        "name": "commitmentSignature",
        "type": "bytes"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "",
        "type": "address"
      }
    ],
    "name": "commitmentsCount",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "string",
        "name": "_txnHash",
        "type": "string"
      },
      {
        "internalType": "uint64",
        "name": "_bid",
        "type": "uint64"
      },
      {
        "internalType": "uint64",
        "name": "_blockNumber",
        "type": "uint64"
      }
    ],
    "name": "getBidHash",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "commitmentHash",
        "type": "bytes32"
      }
    ],
    "name": "getCommitment",
    "outputs": [
      {
        "components": [
          {
            "internalType": "bool",
            "name": "commitmentUsed",
            "type": "bool"
          },
          {
            "internalType": "address",
            "name": "bidder",
            "type": "address"
          },
          {
            "internalType": "address",
            "name": "commiter",
            "type": "address"
          },
          {
            "internalType": "uint64",
            "name": "bid",
            "type": "uint64"
          },
          {
            "internalType": "uint64",
            "name": "blockNumber",
            "type": "uint64"
          },
          {
            "internalType": "bytes32",
            "name": "bidHash",
            "type": "bytes32"
          },
          {
            "internalType": "string",
            "name": "txnHash",
            "type": "string"
          },
          {
            "internalType": "string",
            "name": "commitmentHash",
            "type": "string"
          },
          {
            "internalType": "bytes",
            "name": "bidSignature",
            "type": "bytes"
          },
          {
            "internalType": "bytes",
            "name": "commitmentSignature",
            "type": "bytes"
          }
        ],
        "internalType": "struct PreConfCommitmentStore.PreConfCommitment",
        "name": "",
        "type": "tuple"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "components": [
          {
            "internalType": "bool",
            "name": "commitmentUsed",
            "type": "bool"
          },
          {
            "internalType": "address",
            "name": "bidder",
            "type": "address"
          },
          {
            "internalType": "address",
            "name": "commiter",
            "type": "address"
          },
          {
            "internalType": "uint64",
            "name": "bid",
            "type": "uint64"
          },
          {
            "internalType": "uint64",
            "name": "blockNumber",
            "type": "uint64"
          },
          {
            "internalType": "bytes32",
            "name": "bidHash",
            "type": "bytes32"
          },
          {
            "internalType": "string",
            "name": "txnHash",
            "type": "string"
          },
          {
            "internalType": "string",
            "name": "commitmentHash",
            "type": "string"
          },
          {
            "internalType": "bytes",
            "name": "bidSignature",
            "type": "bytes"
          },
          {
            "internalType": "bytes",
            "name": "commitmentSignature",
            "type": "bytes"
          }
        ],
        "internalType": "struct PreConfCommitmentStore.PreConfCommitment",
        "name": "commitment",
        "type": "tuple"
      }
    ],
    "name": "getCommitmentIndex",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "pure",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "commiter",
        "type": "address"
      }
    ],
    "name": "getCommitmentsByCommitter",
    "outputs": [
      {
        "internalType": "bytes32[]",
        "name": "",
        "type": "bytes32[]"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "string",
        "name": "_txnHash",
        "type": "string"
      },
      {
        "internalType": "uint64",
        "name": "_bid",
        "type": "uint64"
      },
      {
        "internalType": "uint64",
        "name": "_blockNumber",
        "type": "uint64"
      },
      {
        "internalType": "bytes32",
        "name": "_bidHash",
        "type": "bytes32"
      },
      {
        "internalType": "string",
        "name": "_bidSignature",
        "type": "string"
      }
    ],
    "name": "getPreConfHash",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "commitmentIndex",
        "type": "bytes32"
      }
    ],
    "name": "initateReward",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "commitmentIndex",
        "type": "bytes32"
      }
    ],
    "name": "initiateSlash",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "lastProcessedBlock",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "oracle",
    "outputs": [
      {
        "internalType": "address",
        "name": "",
        "type": "address"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "owner",
    "outputs": [
      {
        "internalType": "address",
        "name": "",
        "type": "address"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "",
        "type": "address"
      },
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "name": "providerCommitments",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "providerRegistry",
    "outputs": [
      {
        "internalType": "contract IProviderRegistry",
        "name": "",
        "type": "address"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "renounceOwnership",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "bid",
        "type": "uint64"
      },
      {
        "internalType": "uint64",
        "name": "blockNumber",
        "type": "uint64"
      },
      {
        "internalType": "string",
        "name": "txnHash",
        "type": "string"
      },
      {
        "internalType": "string",
        "name": "commitmentHash",
        "type": "string"
      },
      {
        "internalType": "bytes",
        "name": "bidSignature",
        "type": "bytes"
      },
      {
        "internalType": "bytes",
        "name": "commitmentSignature",
        "type": "bytes"
      }
    ],
    "name": "storeCommitment",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "commitmentIndex",
        "type": "bytes32"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "newOwner",
        "type": "address"
      }
    ],
    "name": "transferOwnership",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "newOracle",
        "type": "address"
      }
    ],
    "name": "updateOracle",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "newProviderRegistry",
        "type": "address"
      }
    ],
    "name": "updateProviderRegistry",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "newUserRegistry",
        "type": "address"
      }
    ],
    "name": "updateUserRegistry",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "userRegistry",
    "outputs": [
      {
        "internalType": "contract IUserRegistry",
        "name": "",
        "type": "address"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "bid",
        "type": "uint64"
      },
      {
        "internalType": "uint64",
        "name": "blockNumber",
        "type": "uint64"
      },
      {
        "internalType": "string",
        "name": "txnHash",
        "type": "string"
      },
      {
        "internalType": "bytes",
        "name": "bidSignature",
        "type": "bytes"
      }
    ],
    "name": "verifyBid",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "messageDigest",
        "type": "bytes32"
      },
      {
        "internalType": "address",
        "name": "recoveredAddress",
        "type": "address"
      },
      {
        "internalType": "uint256",
        "name": "stake",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "string",
        "name": "txnHash",
        "type": "string"
      },
      {
        "internalType": "uint64",
        "name": "bid",
        "type": "uint64"
      },
      {
        "internalType": "uint64",
        "name": "blockNumber",
        "type": "uint64"
      },
      {
        "internalType": "bytes32",
        "name": "bidHash",
        "type": "bytes32"
      },
      {
        "internalType": "bytes",
        "name": "bidSignature",
        "type": "bytes"
      },
      {
        "internalType": "bytes",
        "name": "commitmentSignature",
        "type": "bytes"
      }
    ],
    "name": "verifyPreConfCommitment",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "preConfHash",
        "type": "bytes32"
      },
      {
        "internalType": "address",
        "name": "commiterAddress",
        "type": "address"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "stateMutability": "payable",
    "type": "receive"
  }
]
`
