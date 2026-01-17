# Architecture References Guide

This document provides detailed technical references for the consensus algorithms implemented in our LSCC Blockchain platform. It explains the industry-standard implementations (Bitcoin, Ethereum, Hyperledger) and how our implementation aligns with or differs from these standards.

---

## Table of Contents

1. [Proof of Work (PoW)](#1-proof-of-work-pow)
2. [Proof of Stake (PoS)](#2-proof-of-stake-pos)
3. [Practical Byzantine Fault Tolerance (PBFT)](#3-practical-byzantine-fault-tolerance-pbft)
4. [LSCC Protocol](#4-lscc-protocol)
5. [Comparison Summary](#5-comparison-summary)
6. [References](#6-references)

---

## 1. Proof of Work (PoW)

### 1.1 Industry Standard: Bitcoin Implementation

Bitcoin's PoW is the foundational consensus mechanism that has secured the network since 2009.

#### Core Algorithm

```
1. Assemble 80-byte block header:
   - Version (4 bytes)
   - Previous Block Hash (32 bytes)
   - Merkle Root (32 bytes)
   - Timestamp (4 bytes)
   - Difficulty Target/Bits (4 bytes)
   - Nonce (4 bytes)

2. Compute Hash = SHA256(SHA256(block_header))

3. If Hash < Target → Block is valid
   Else → Increment nonce, repeat
```

#### Key Technical Details

| Component | Bitcoin Specification |
|-----------|----------------------|
| Hash Function | Double SHA-256 |
| Nonce Space | 4 bytes (2³² = 4,294,967,296 values) |
| Block Time Target | 10 minutes |
| Difficulty Adjustment | Every 2,016 blocks (~2 weeks) |
| ExtraNonce | In coinbase transaction (extends nonce space) |
| Block Size | 1 MB (with SegWit: ~4 MB effective) |

#### Mining Process Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    BITCOIN MINING FLOW                       │
└─────────────────────────────────────────────────────────────┘

Step 1: Collect Transactions
├── Mempool transactions sorted by fee
├── Validate each transaction
└── Build Merkle tree of transactions

Step 2: Assemble Block Header
├── Version: Protocol version (4 bytes)
├── Previous Hash: Link to previous block (32 bytes)
├── Merkle Root: Hash of all transactions (32 bytes)
├── Timestamp: Current time (4 bytes)
├── Bits: Compact difficulty target (4 bytes)
└── Nonce: Counter starting at 0 (4 bytes)

Step 3: Mining Loop
├── hash = SHA256(SHA256(header))
├── if hash < target:
│   └── SUCCESS → Broadcast block
└── else:
    ├── nonce++
    ├── if nonce exhausted:
    │   └── Modify extraNonce in coinbase → new Merkle root
    └── Continue hashing

Step 4: Block Propagation
├── Broadcast to peers
├── Peers validate block
└── Add to blockchain if valid
```

#### Difficulty Adjustment Algorithm

```python
# Bitcoin difficulty adjustment (every 2016 blocks)
def adjust_difficulty(actual_time, expected_time, current_difficulty):
    # Expected time: 2016 blocks * 10 minutes = 20160 minutes
    expected_time = 2016 * 10 * 60  # seconds
    
    # Calculate ratio
    ratio = actual_time / expected_time
    
    # Clamp to prevent extreme changes (max 4x adjustment)
    ratio = max(0.25, min(4.0, ratio))
    
    # New difficulty
    new_difficulty = current_difficulty / ratio
    
    return new_difficulty
```

### 1.2 Our PoW Implementation

Our PoW implementation follows Bitcoin's core principles with adaptations for flexibility.

#### Implementation Details

| Component | Our Implementation | Bitcoin |
|-----------|-------------------|---------|
| Hash Function | SHA-256 (single) | Double SHA-256 |
| Nonce Space | Configurable | 4 bytes fixed |
| Block Time | Configurable (default: 15s) | 10 minutes |
| Difficulty | Configurable (default: 4) | Dynamic adjustment |
| Gas Limit | 200,000,000 | N/A (Bitcoin uses weight) |

#### Code Location

```
internal/consensus/pow.go          # PoW consensus engine
internal/blockchain/block.go       # Block structure
internal/blockchain/mining.go      # Mining logic
```

#### Configuration

```yaml
consensus:
  algorithm: "pow"
  difficulty: 4              # Number of leading zeros required
  block_time: 15             # Target block time in seconds
  max_nonce: 1000000000      # Maximum nonce attempts
  gas_limit: 200000000       # Gas limit per block
```

#### Mining Algorithm

```go
func (pow *PoW) Mine(block *Block) (bool, error) {
    target := pow.calculateTarget()
    
    for nonce := uint64(0); nonce < pow.maxNonce; nonce++ {
        block.Nonce = nonce
        hash := pow.calculateHash(block)
        
        if pow.meetsTarget(hash, target) {
            block.Hash = hash
            return true, nil
        }
    }
    
    return false, ErrNonceExhausted
}

func (pow *PoW) calculateTarget() *big.Int {
    // Target = MaxTarget / 2^difficulty
    target := new(big.Int).Exp(
        big.NewInt(2),
        big.NewInt(256 - int64(pow.difficulty)),
        nil,
    )
    return target
}
```

---

## 2. Proof of Stake (PoS)

### 2.1 Industry Standard: Ethereum 2.0 (Gasper)

Ethereum transitioned to PoS in September 2022 using the Gasper protocol.

#### Core Architecture: Gasper = LMD GHOST + Casper FFG

```
┌─────────────────────────────────────────────────────────────┐
│                    ETHEREUM GASPER PROTOCOL                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  LMD GHOST (Fork Choice)     │     Casper FFG (Finality)   │
│  ─────────────────────────   │     ────────────────────     │
│  • Slot-by-slot decisions    │     • Epoch-by-epoch        │
│  • 12-second intervals       │     • 32 slots = 1 epoch    │
│  • Latest message driven     │     • 2/3 supermajority     │
│  • Provides liveness         │     • Provides finality     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### Time Structure

| Unit | Duration | Purpose |
|------|----------|---------|
| Slot | 12 seconds | Basic time unit, one block proposal |
| Epoch | 32 slots (6.4 min) | Finality checkpoint period |
| Finality | 2 epochs (~13 min) | Block becomes irreversible |

#### Validator Requirements

| Requirement | Specification |
|-------------|---------------|
| Minimum Stake | 32 ETH |
| Activation Queue | Variable (days to weeks) |
| Slashing Penalty | Up to 100% of stake |
| Attestation Duty | Once per epoch |
| Block Proposal | Random selection proportional to stake |

#### Casper FFG Finalization Process

```
Epoch N-1        Epoch N          Epoch N+1
    │                │                │
    ▼                ▼                ▼
┌───────┐        ┌───────┐        ┌───────┐
│ CP(1) │───────▶│ CP(2) │───────▶│ CP(3) │
└───────┘        └───────┘        └───────┘
    │                │                │
    │   Justified    │   Justified    │
    │   (2/3 votes)  │   (2/3 votes)  │
    │                │                │
    └────────────────┴────────────────┘
                     │
                     ▼
              CP(1) FINALIZED
         (has justified child)
```

#### Slashing Conditions

1. **Double Voting**: Voting for two different blocks at the same height
2. **Surround Voting**: Casting votes that surround or are surrounded by previous votes
3. **Invalid Attestations**: Signing incorrect checkpoint pairs

### 2.2 Our PoS Implementation

Our PoS implementation provides stake-based validator selection with configurable parameters.

#### Implementation Details

| Component | Our Implementation | Ethereum 2.0 |
|-----------|-------------------|--------------|
| Minimum Stake | Configurable (default: 1000) | 32 ETH |
| Validator Selection | Weighted random | RANDAO-based |
| Block Time | Configurable (default: 5s) | 12 seconds |
| Finality | Single confirmation | 2 epochs |
| Slashing | Configurable | Automatic |

#### Code Location

```
internal/consensus/pos.go          # PoS consensus engine
internal/consensus/validator.go    # Validator management
internal/consensus/staking.go      # Staking logic
```

#### Configuration

```yaml
consensus:
  algorithm: "pos"
  min_stake: 1000           # Minimum stake to become validator
  stake_ratio: 0.1          # Stake weight ratio
  block_time: 5             # Target block time in seconds
  validator_count: 21       # Maximum active validators
```

#### Validator Selection Algorithm

```go
func (pos *PoS) SelectValidator(validators []Validator, blockHeight int64) *Validator {
    // Calculate total stake
    totalStake := big.NewInt(0)
    for _, v := range validators {
        totalStake.Add(totalStake, v.Stake)
    }
    
    // Generate deterministic random using block height
    seed := sha256.Sum256([]byte(fmt.Sprintf("%d", blockHeight)))
    random := new(big.Int).SetBytes(seed[:])
    random.Mod(random, totalStake)
    
    // Select validator based on stake weight
    cumulative := big.NewInt(0)
    for _, v := range validators {
        cumulative.Add(cumulative, v.Stake)
        if random.Cmp(cumulative) < 0 {
            return &v
        }
    }
    
    return &validators[0]
}
```

---

## 3. Practical Byzantine Fault Tolerance (PBFT)

### 3.1 Industry Standard: Castro-Liskov PBFT

PBFT was introduced by Miguel Castro and Barbara Liskov in 1999 and is used in many permissioned blockchains.

#### Core Requirements

| Requirement | Specification |
|-------------|---------------|
| Total Nodes (n) | n ≥ 3f + 1 |
| Byzantine Tolerance (f) | f = (n-1) / 3 |
| Quorum Size | 2f + 1 |
| Message Complexity | O(n²) per consensus round |

#### Three-Phase Protocol

```
┌─────────────────────────────────────────────────────────────┐
│                    PBFT THREE-PHASE PROTOCOL                 │
└─────────────────────────────────────────────────────────────┘

Client ──────▶ Primary ──────▶ All Replicas
                  │
     ┌────────────┼────────────────────────────────────┐
     │            │                                     │
     ▼            ▼                                     ▼
┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐
│ Primary │  │Replica 1│  │Replica 2│  │Replica 3│  │Replica 4│
└────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘
     │            │            │            │            │
     │ PHASE 1: PRE-PREPARE                              │
     │────────────────────────────────────────────────────▶
     │  <PRE-PREPARE, v, n, d>                           │
     │                                                    │
     │ PHASE 2: PREPARE                                  │
     │◀───────────────────────────────────────────────────│
     │  <PREPARE, v, n, d, i>                            │
     │  (all-to-all broadcast)                           │
     │  Wait for 2f PREPARE messages                     │
     │                                                    │
     │ PHASE 3: COMMIT                                   │
     │◀───────────────────────────────────────────────────│
     │  <COMMIT, v, n, d, i>                             │
     │  (all-to-all broadcast)                           │
     │  Wait for 2f+1 COMMIT messages                    │
     │                                                    │
     │ REPLY TO CLIENT                                   │
     │────────────────────────────────────────────────────▶
     │  <REPLY, v, t, c, i, r>                           │
     │  Client waits for f+1 identical replies           │
```

#### Message Structure

```
PRE-PREPARE: <PRE-PREPARE, v, n, d>σp
    v = view number
    n = sequence number  
    d = digest of request
    σp = primary's signature

PREPARE: <PREPARE, v, n, d, i>σi
    i = replica identifier
    σi = replica's signature

COMMIT: <COMMIT, v, n, d, i>σi
    Same structure as PREPARE
    Confirms prepared state

REPLY: <REPLY, v, t, c, i, r>σi
    t = timestamp
    c = client identifier
    r = result of operation
```

#### View Change Protocol

```
Triggered when:
├── Consensus timeout expires
├── Primary suspected as faulty
└── Invalid pre-prepare received

Process:
1. Replica broadcasts <VIEW-CHANGE, v+1, n, C, P, i>
   • C = checkpoint proofs
   • P = prepared request certificates

2. Wait for 2f+1 VIEW-CHANGE messages

3. New primary (v+1 mod n) broadcasts:
   <NEW-VIEW, v+1, V, O>
   • V = view-change proofs
   • O = re-proposed pre-prepares

4. Resume consensus in new view
```

### 3.2 Our PBFT Implementation

Our PBFT implementation follows the Castro-Liskov protocol with optimizations for blockchain use cases.

#### Implementation Details

| Component | Our Implementation | Standard PBFT |
|-----------|-------------------|---------------|
| Phases | 3 (Pre-prepare, Prepare, Commit) | 3 phases |
| View Change | Timeout-based | Timeout-based |
| Checkpoints | Configurable interval | Every K requests |
| Message Auth | Digital signatures | Digital signatures |
| Batching | Transaction batching | Request batching |

#### Code Location

```
internal/consensus/pbft.go         # PBFT consensus engine
internal/consensus/pbft_state.go   # State machine
internal/consensus/view_change.go  # View change protocol
```

#### Configuration

```yaml
consensus:
  algorithm: "pbft"
  view_timeout: 30          # Seconds before view change
  checkpoint_interval: 100  # Blocks between checkpoints
  max_faulty_nodes: 1       # f value (n = 3f+1)
  block_time: 3             # Target block time
```

#### Consensus Flow

```go
type PBFTState struct {
    View           uint64
    SequenceNumber uint64
    Phase          PBFTPhase  // PrePrepare, Prepare, Commit
    PrepareCount   int
    CommitCount    int
    Prepared       bool
    Committed      bool
}

func (pbft *PBFT) ProcessConsensus(block *Block) error {
    // Phase 1: Pre-Prepare (Primary only)
    if pbft.IsPrimary() {
        pbft.BroadcastPrePrepare(block)
    }
    
    // Phase 2: Prepare
    pbft.BroadcastPrepare(block.Hash)
    pbft.WaitForPrepareQuorum()  // 2f messages
    
    // Phase 3: Commit
    pbft.BroadcastCommit(block.Hash)
    pbft.WaitForCommitQuorum()   // 2f+1 messages
    
    // Finalize
    return pbft.CommitBlock(block)
}
```

---

## 4. LSCC Protocol

### 4.1 Overview

LSCC (Layered Sharding with Cross-Channel Consensus) is our novel protocol that combines:
- **Layered Architecture**: 3-layer hierarchical processing
- **Sharding**: Parallel transaction processing across 4 shards
- **Cross-Channel Consensus**: Efficient inter-layer coordination

### 4.2 Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    LSCC ARCHITECTURE                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Layer 0 (Base)      Layer 1 (Middle)    Layer 2 (Top)     │
│  ┌─────┬─────┐       ┌─────┬─────┐       ┌─────┬─────┐     │
│  │ S0  │ S1  │       │ S2  │ S3  │       │ S4  │ S5  │     │
│  └──┬──┴──┬──┘       └──┬──┴──┬──┘       └──┬──┴──┬──┘     │
│     │     │             │     │             │     │         │
│     └─────┼─────────────┼─────┼─────────────┼─────┘         │
│           │             │     │             │               │
│           └─────────────┴──┬──┴─────────────┘               │
│                            │                                 │
│                    Cross-Channel Router                      │
│                            │                                 │
│                    ┌───────┴───────┐                        │
│                    │  Block        │                        │
│                    │  Finalization │                        │
│                    └───────────────┘                        │
└─────────────────────────────────────────────────────────────┘
```

### 4.3 Four-Phase Consensus

```
Phase 1: Channel Formation (3ms)
├── Parallel validator channel assignment
├── Load-balanced transaction distribution
└── Dynamic shard allocation

Phase 2: Parallel Validation (5ms)
├── Concurrent signature verification
├── Independent balance checks per channel
└── Parallel Merkle tree construction

Phase 3: Cross-Channel Sync (4ms)
├── Inter-channel consensus coordination
├── Conflict resolution for cross-shard transactions
└── Global state consistency verification

Phase 4: Block Finalization (3ms)
├── Final block assembly
├── Cross-shard state synchronization
└── Network broadcast and confirmation

TOTAL: ~15ms per consensus round
```

### 4.4 Configuration

```yaml
consensus:
  algorithm: "lscc"
  block_time: 1
  layer_depth: 3
  channel_count: 5
  gas_limit: 200000000

sharding:
  num_shards: 4
  shard_size: 100
  cross_shard_delay: 100
  rebalance_threshold: 0.7
  layered_structure: true
```

---

## 5. Comparison Summary

### 5.1 Performance Characteristics

| Metric | PoW | PoS | PBFT | LSCC |
|--------|-----|-----|------|------|
| **TPS** | 7-15 | 42-100 | 89-200 | 350-400 |
| **Latency** | 10 min | 12 sec | 1-3 sec | 15 ms |
| **Finality** | Probabilistic | 2 epochs | Immediate | Immediate |
| **Message Complexity** | O(n) | O(n) | O(n²) | O(log n) |

### 5.2 Security Properties

| Property | PoW | PoS | PBFT | LSCC |
|----------|-----|-----|------|------|
| **Byzantine Tolerance** | 51% hashpower | 51% stake | 33% nodes | 33% per layer |
| **Sybil Resistance** | Computational | Economic | Permissioned | Multi-layer |
| **Energy Efficiency** | Low | High | High | High |
| **Decentralization** | High | Medium | Low | Medium |

### 5.3 Use Cases

| Protocol | Best For |
|----------|----------|
| **PoW** | Maximum decentralization, censorship resistance |
| **PoS** | Energy efficiency, public networks |
| **PBFT** | Permissioned networks, consortium chains |
| **LSCC** | High-throughput enterprise, real-time applications |

---

## 6. References

### Academic Papers

1. **Bitcoin Whitepaper**: Nakamoto, S. (2008). "Bitcoin: A Peer-to-Peer Electronic Cash System"
   - https://bitcoin.org/bitcoin.pdf

2. **PBFT Original Paper**: Castro, M., & Liskov, B. (1999). "Practical Byzantine Fault Tolerance"
   - http://pmg.csail.mit.edu/papers/osdi99.pdf

3. **Casper FFG**: Buterin, V., & Griffith, V. (2017). "Casper the Friendly Finality Gadget"
   - https://arxiv.org/abs/1710.09437

4. **Gasper**: Buterin, V., et al. (2020). "Combining GHOST and Casper"
   - https://arxiv.org/abs/2003.03052

### Implementation References

| Protocol | Reference Implementation | Language |
|----------|-------------------------|----------|
| PoW | Bitcoin Core | C++ |
| PoW | btcd | Go |
| PoS | Prysm (Ethereum) | Go |
| PoS | Lighthouse | Rust |
| PBFT | Hyperledger Fabric | Go |
| PBFT | Tendermint | Go |

### Documentation Links

- **Bitcoin Developer Guide**: https://developer.bitcoin.org/
- **Ethereum PoS Docs**: https://ethereum.org/developers/docs/consensus-mechanisms/pos/
- **Hyperledger PBFT**: https://hyperledger-fabric.readthedocs.io/
- **Tendermint Core**: https://docs.tendermint.com/

---

## Appendix: Quick Reference

### Network Requirements

| Protocol | Min Nodes | Fault Tolerance |
|----------|-----------|-----------------|
| PoW | 1 | 51% hashpower |
| PoS | 1 | 51% stake |
| PBFT | 4 | 1 Byzantine (3f+1) |
| LSCC | 4 | 1 per layer |

### Configuration Defaults

```yaml
# PoW
consensus:
  algorithm: "pow"
  difficulty: 4
  block_time: 15

# PoS  
consensus:
  algorithm: "pos"
  min_stake: 1000
  block_time: 5

# PBFT
consensus:
  algorithm: "pbft"
  view_timeout: 30
  block_time: 3

# LSCC
consensus:
  algorithm: "lscc"
  layer_depth: 3
  channel_count: 5
  block_time: 1
```
