package consensus

import (
        "bytes"
        "crypto/sha256"
        "encoding/binary"
        "encoding/hex"
        "fmt"
        "lscc-blockchain/config"
        "lscc-blockchain/internal/utils"
        "lscc-blockchain/pkg/types"
        "math/big"
        "sync"
        "time"

        "github.com/sirupsen/logrus"
)

const (
        MaxNonce              = uint32(0xFFFFFFFF)
        DifficultyAdjInterval = 2016
        TargetBlockTime       = 600
        MaxFutureBlockTime    = 2 * 60 * 60
        MedianTimePastBlocks  = 11
        InitialBlockReward    = 5000000000
        HalvingInterval       = 210000
        MaxBlockSize          = 1000000
        MinDifficultyBits     = 0x1d00ffff
)

type BlockHeader struct {
        Version       int32
        PrevBlockHash [32]byte
        MerkleRoot    [32]byte
        Timestamp     uint32
        Bits          uint32
        Nonce         uint32
}

func (h *BlockHeader) Serialize() []byte {
        buf := new(bytes.Buffer)
        binary.Write(buf, binary.LittleEndian, h.Version)
        buf.Write(h.PrevBlockHash[:])
        buf.Write(h.MerkleRoot[:])
        binary.Write(buf, binary.LittleEndian, h.Timestamp)
        binary.Write(buf, binary.LittleEndian, h.Bits)
        binary.Write(buf, binary.LittleEndian, h.Nonce)
        return buf.Bytes()
}

type CoinbaseTransaction struct {
        BlockHeight uint32
        ExtraNonce  uint64
        Outputs     []CoinbaseOutput
        ScriptSig   []byte
}

type CoinbaseOutput struct {
        Value        int64
        ScriptPubKey []byte
}

type BitcoinPoW struct {
        config           *config.Config
        logger           *utils.Logger
        difficulty       *big.Int
        bits             uint32
        state            *types.ConsensusState
        mu               sync.RWMutex
        hashRate         float64
        totalHashes      uint64
        blocksFound      uint64
        startTime        time.Time
        metrics          map[string]interface{}
        blockTimes       []time.Time
        lastAdjustment   int64
        currentHeight    int64
        extraNonce       uint64
        miningActive     bool
        stopMining       chan struct{}
}

func NewBitcoinPoW(cfg *config.Config, logger *utils.Logger) (*BitcoinPoW, error) {
        startTime := time.Now()

        logger.LogConsensus("bitcoin_pow", "initialize", logrus.Fields{
                "difficulty_bits": cfg.Consensus.Difficulty,
                "block_time":      cfg.Consensus.BlockTime,
                "timestamp":       startTime,
        })

        initialBits := uint32(0x1d00ffff)
        if cfg.Consensus.Difficulty > 0 && cfg.Consensus.Difficulty < 32 {
                initialBits = difficultyToBits(cfg.Consensus.Difficulty)
        }

        pow := &BitcoinPoW{
                config:         cfg,
                logger:         logger,
                bits:           initialBits,
                difficulty:     bitsToTarget(initialBits),
                startTime:      startTime,
                metrics:        make(map[string]interface{}),
                blockTimes:     make([]time.Time, 0, DifficultyAdjInterval),
                lastAdjustment: 0,
                currentHeight:  0,
                extraNonce:     0,
                miningActive:   false,
                stopMining:     make(chan struct{}),
                state: &types.ConsensusState{
                        Algorithm:    "bitcoin_pow",
                        Round:        0,
                        View:         0,
                        Phase:        "ready",
                        Validators:   make([]*types.Validator, 0),
                        Votes:        make(map[string]interface{}),
                        LastDecision: startTime,
                        Performance:  make(map[string]float64),
                },
        }

        pow.updateMetrics()

        logger.LogConsensus("bitcoin_pow", "initialized", logrus.Fields{
                "difficulty_bits": pow.bits,
                "target":          pow.difficulty.Text(16),
                "timestamp":       time.Now().UTC(),
        })

        return pow, nil
}

func DoubleSHA256(data []byte) [32]byte {
        first := sha256.Sum256(data)
        return sha256.Sum256(first[:])
}

func DoubleSHA256Hex(data []byte) string {
        hash := DoubleSHA256(data)
        reversed := ReverseBytes(hash[:])
        return hex.EncodeToString(reversed)
}

func ReverseBytes(data []byte) []byte {
        reversed := make([]byte, len(data))
        for i := 0; i < len(data); i++ {
                reversed[i] = data[len(data)-1-i]
        }
        return reversed
}

func HashToTarget(hash [32]byte) *big.Int {
        reversed := ReverseBytes(hash[:])
        return new(big.Int).SetBytes(reversed)
}

func ReverseHash32(hash [32]byte) [32]byte {
        var reversed [32]byte
        for i := 0; i < 32; i++ {
                reversed[i] = hash[31-i]
        }
        return reversed
}

func bitsToTarget(bits uint32) *big.Int {
        exponent := bits >> 24
        mantissa := bits & 0x007fffff

        target := big.NewInt(int64(mantissa))

        if exponent <= 3 {
                target.Rsh(target, uint(8*(3-exponent)))
        } else {
                target.Lsh(target, uint(8*(exponent-3)))
        }

        maxTarget := new(big.Int)
        maxTarget.SetString("00000000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 16)
        if target.Cmp(maxTarget) > 0 {
                return maxTarget
        }

        return target
}

func targetToBits(target *big.Int) uint32 {
        bytes := target.Bytes()
        if len(bytes) == 0 {
                return 0
        }
        exponent := uint32(len(bytes))
        var mantissa uint32
        if len(bytes) >= 3 {
                mantissa = uint32(bytes[0])<<16 | uint32(bytes[1])<<8 | uint32(bytes[2])
        } else if len(bytes) == 2 {
                mantissa = uint32(bytes[0])<<16 | uint32(bytes[1])<<8
        } else {
                mantissa = uint32(bytes[0]) << 16
        }
        if mantissa&0x00800000 != 0 {
                mantissa >>= 8
                exponent++
        }
        return (exponent << 24) | mantissa
}

func difficultyToBits(difficulty int) uint32 {
        maxTarget := bitsToTarget(0x1d00ffff)
        newTarget := new(big.Int).Div(maxTarget, big.NewInt(int64(1<<uint(difficulty*4))))
        return targetToBits(newTarget)
}

func (pow *BitcoinPoW) createBlockHeader(block *types.Block) *BlockHeader {
        var prevHash [32]byte
        if prevBytes, err := hex.DecodeString(block.PreviousHash); err == nil && len(prevBytes) == 32 {
                reversed := ReverseBytes(prevBytes)
                copy(prevHash[:], reversed)
        }

        var merkleRoot [32]byte
        if mrBytes, err := hex.DecodeString(block.MerkleRoot); err == nil && len(mrBytes) == 32 {
                reversed := ReverseBytes(mrBytes)
                copy(merkleRoot[:], reversed)
        }

        return &BlockHeader{
                Version:       1,
                PrevBlockHash: prevHash,
                MerkleRoot:    merkleRoot,
                Timestamp:     uint32(block.Timestamp.Unix()),
                Bits:          pow.bits,
                Nonce:         uint32(block.Nonce),
        }
}

func (pow *BitcoinPoW) ProcessBlock(block *types.Block, validators []*types.Validator) (bool, error) {
        startTime := time.Now()
        pow.mu.Lock()
        defer pow.mu.Unlock()

        pow.logger.LogConsensus("bitcoin_pow", "process_block", logrus.Fields{
                "block_hash":  block.Hash,
                "block_index": block.Index,
                "bits":        pow.bits,
                "validator":   block.Validator,
                "tx_count":    len(block.Transactions),
                "timestamp":   startTime,
        })

        pow.state.Round = block.Index
        pow.state.Phase = "mining"
        pow.state.Validators = validators

        pow.miningActive = true
        miningStart := time.Now()
        success, hashAttempts, err := pow.mineBlockBitcoin(block)
        miningDuration := time.Since(miningStart)
        pow.miningActive = false

        if err != nil {
                pow.logger.LogError("consensus", "mine_block_bitcoin", err, logrus.Fields{
                        "block_hash": block.Hash,
                        "timestamp":  time.Now().UTC(),
                })
                return false, fmt.Errorf("bitcoin mining failed: %w", err)
        }

        if !success {
                pow.logger.LogConsensus("bitcoin_pow", "mining_failed", logrus.Fields{
                        "block_hash":      block.Hash,
                        "hash_attempts":   hashAttempts,
                        "mining_duration": miningDuration.Milliseconds(),
                        "bits":            pow.bits,
                        "timestamp":       time.Now().UTC(),
                })
                return false, nil
        }

        pow.totalHashes += hashAttempts
        pow.blocksFound++
        pow.hashRate = float64(pow.totalHashes) / time.Since(pow.startTime).Seconds()
        pow.currentHeight = block.Index

        pow.blockTimes = append(pow.blockTimes, time.Now())
        if len(pow.blockTimes) > DifficultyAdjInterval {
                pow.blockTimes = pow.blockTimes[1:]
        }

        if pow.currentHeight > 0 && pow.currentHeight%DifficultyAdjInterval == 0 {
                pow.adjustDifficultyBitcoin()
        }

        pow.state.Phase = "completed"
        pow.state.LastDecision = time.Now()

        totalDuration := time.Since(startTime)

        pow.state.Performance["mining_duration"] = miningDuration.Seconds()
        pow.state.Performance["total_duration"] = totalDuration.Seconds()
        pow.state.Performance["hash_rate"] = pow.hashRate
        pow.state.Performance["hash_attempts"] = float64(hashAttempts)

        pow.updateMetrics()

        pow.logger.LogConsensus("bitcoin_pow", "block_processed", logrus.Fields{
                "block_hash":      block.Hash,
                "block_index":     block.Index,
                "nonce":           block.Nonce,
                "hash_attempts":   hashAttempts,
                "mining_duration": miningDuration.Milliseconds(),
                "total_duration":  totalDuration.Milliseconds(),
                "hash_rate":       pow.hashRate,
                "bits":            pow.bits,
                "blocks_found":    pow.blocksFound,
                "timestamp":       time.Now().UTC(),
        })

        return true, nil
}

func (pow *BitcoinPoW) mineBlockBitcoin(block *types.Block) (bool, uint64, error) {
        header := pow.createBlockHeader(block)
        target := pow.difficulty
        var hashAttempts uint64 = 0

        pow.logger.LogConsensus("bitcoin_pow", "start_mining", logrus.Fields{
                "block_index":    block.Index,
                "target":         target.Text(16)[:16] + "...",
                "bits":           pow.bits,
                "initial_nonce":  header.Nonce,
                "extra_nonce":    pow.extraNonce,
                "timestamp":      time.Now().UTC(),
        })

        for pow.extraNonce < 0xFFFFFFFFFFFFFFFF {
                header.Nonce = 0

                for header.Nonce <= MaxNonce {
                        select {
                        case <-pow.stopMining:
                                return false, hashAttempts, nil
                        default:
                        }

                        hashAttempts++
                        headerBytes := header.Serialize()
                        hash := DoubleSHA256(headerBytes)

                        hashInt := HashToTarget(hash)

                        if hashInt.Cmp(target) <= 0 {
                                block.Nonce = int64(header.Nonce)
                                block.Hash = DoubleSHA256Hex(headerBytes)

                                pow.logger.LogConsensus("bitcoin_pow", "mining_success", logrus.Fields{
                                        "block_hash":    block.Hash,
                                        "block_index":   block.Index,
                                        "final_nonce":   header.Nonce,
                                        "extra_nonce":   pow.extraNonce,
                                        "hash_attempts": hashAttempts,
                                        "bits":          pow.bits,
                                        "timestamp":     time.Now().UTC(),
                                })

                                return true, hashAttempts, nil
                        }

                        if hashAttempts%100000 == 0 {
                                pow.logger.LogConsensus("bitcoin_pow", "mining_progress", logrus.Fields{
                                        "block_index":   block.Index,
                                        "hash_attempts": hashAttempts,
                                        "current_nonce": header.Nonce,
                                        "extra_nonce":   pow.extraNonce,
                                        "hash_rate":     float64(hashAttempts) / time.Since(pow.startTime).Seconds(),
                                        "timestamp":     time.Now().UTC(),
                                })
                        }

                        header.Nonce++
                }

                pow.extraNonce++
                pow.updateMerkleRootWithExtraNonce(block, pow.extraNonce)
                header = pow.createBlockHeader(block)

                pow.logger.LogConsensus("bitcoin_pow", "nonce_exhausted", logrus.Fields{
                        "block_index":    block.Index,
                        "new_extra_nonce": pow.extraNonce,
                        "hash_attempts":  hashAttempts,
                        "timestamp":      time.Now().UTC(),
                })
        }

        return false, hashAttempts, nil
}

func (pow *BitcoinPoW) updateMerkleRootWithExtraNonce(block *types.Block, extraNonce uint64) {
        if len(block.Transactions) == 0 {
                block.MerkleRoot = fmt.Sprintf("%064x", extraNonce)
                return
        }

        coinbaseData := fmt.Sprintf("coinbase:%d:%d", block.Index, extraNonce)
        coinbaseHash := sha256.Sum256([]byte(coinbaseData))

        var txHashes [][32]byte
        txHashes = append(txHashes, coinbaseHash)

        for _, tx := range block.Transactions {
                txHashBytes, _ := hex.DecodeString(tx.Hash())
                var txHash [32]byte
                copy(txHash[:], txHashBytes)
                txHashes = append(txHashes, txHash)
        }

        for len(txHashes) > 1 {
                if len(txHashes)%2 == 1 {
                        txHashes = append(txHashes, txHashes[len(txHashes)-1])
                }

                var newLevel [][32]byte
                for i := 0; i < len(txHashes); i += 2 {
                        combined := append(txHashes[i][:], txHashes[i+1][:]...)
                        newLevel = append(newLevel, DoubleSHA256(combined))
                }
                txHashes = newLevel
        }

        block.MerkleRoot = hex.EncodeToString(txHashes[0][:])
}

func (pow *BitcoinPoW) adjustDifficultyBitcoin() {
        if len(pow.blockTimes) < 2 {
                return
        }

        blocksToConsider := DifficultyAdjInterval
        if len(pow.blockTimes) < blocksToConsider {
                blocksToConsider = len(pow.blockTimes)
        }

        firstBlockTime := pow.blockTimes[len(pow.blockTimes)-blocksToConsider]
        lastBlockTime := pow.blockTimes[len(pow.blockTimes)-1]
        actualTimeSeconds := int64(lastBlockTime.Sub(firstBlockTime).Seconds())

        expectedTimeSeconds := int64(blocksToConsider) * int64(TargetBlockTime)

        if actualTimeSeconds < expectedTimeSeconds/4 {
                actualTimeSeconds = expectedTimeSeconds / 4
        } else if actualTimeSeconds > expectedTimeSeconds*4 {
                actualTimeSeconds = expectedTimeSeconds * 4
        }

        oldBits := pow.bits
        oldTarget := new(big.Int).Set(pow.difficulty)

        newTarget := new(big.Int).Set(pow.difficulty)
        newTarget.Mul(newTarget, big.NewInt(actualTimeSeconds))
        newTarget.Div(newTarget, big.NewInt(expectedTimeSeconds))

        maxTarget := bitsToTarget(0x1d00ffff)
        if newTarget.Cmp(maxTarget) > 0 {
                newTarget = maxTarget
        }

        minTarget := big.NewInt(1)
        if newTarget.Cmp(minTarget) < 0 {
                newTarget = minTarget
        }

        pow.difficulty = newTarget
        pow.bits = targetToBits(newTarget)
        pow.lastAdjustment = pow.currentHeight

        pow.logger.LogConsensus("bitcoin_pow", "difficulty_adjusted", logrus.Fields{
                "old_bits":           oldBits,
                "new_bits":           pow.bits,
                "old_target":         oldTarget.Text(16),
                "new_target":         newTarget.Text(16),
                "actual_time_sec":    actualTimeSeconds,
                "expected_time_sec":  expectedTimeSeconds,
                "block_height":       pow.currentHeight,
                "timestamp":          time.Now().UTC(),
        })
}

func (pow *BitcoinPoW) ValidateBlock(block *types.Block, validators []*types.Validator) error {
        startTime := time.Now()

        pow.logger.LogConsensus("bitcoin_pow", "validate_block", logrus.Fields{
                "block_hash":  block.Hash,
                "block_index": block.Index,
                "bits":        pow.bits,
                "nonce":       block.Nonce,
                "timestamp":   startTime,
        })

        header := pow.createBlockHeader(block)
        header.Nonce = uint32(block.Nonce)

        headerBytes := header.Serialize()
        hash := DoubleSHA256(headerBytes)
        hashInt := HashToTarget(hash)

        if hashInt.Cmp(pow.difficulty) > 0 {
                return fmt.Errorf("block hash does not meet difficulty target")
        }

        calculatedHash := DoubleSHA256Hex(headerBytes)
        if block.Hash != calculatedHash {
                return fmt.Errorf("block hash verification failed: expected %s, got %s", calculatedHash, block.Hash)
        }

        if err := pow.validateTimestamp(block); err != nil {
                return err
        }

        if err := pow.validateMerkleRoot(block); err != nil {
                return err
        }

        validationDuration := time.Since(startTime)

        pow.logger.LogConsensus("bitcoin_pow", "block_validated", logrus.Fields{
                "block_hash":          block.Hash,
                "block_index":         block.Index,
                "validation_duration": validationDuration.Milliseconds(),
                "timestamp":           time.Now().UTC(),
        })

        return nil
}

func (pow *BitcoinPoW) validateTimestamp(block *types.Block) error {
        now := time.Now()
        maxFuture := now.Add(time.Duration(MaxFutureBlockTime) * time.Second)

        if block.Timestamp.After(maxFuture) {
                return fmt.Errorf("block timestamp too far in future: %v > %v", block.Timestamp, maxFuture)
        }

        return nil
}

func (pow *BitcoinPoW) validateMerkleRoot(block *types.Block) error {
        if len(block.Transactions) == 0 {
                return nil
        }

        return nil
}

func (pow *BitcoinPoW) GetBlockReward(height int64) int64 {
        halvings := height / HalvingInterval
        if halvings >= 64 {
                return 0
        }
        return InitialBlockReward >> uint(halvings)
}

func (pow *BitcoinPoW) CreateCoinbaseTransaction(height int64, minerAddress string) *types.Transaction {
        reward := pow.GetBlockReward(height)

        return &types.Transaction{
                ID:        fmt.Sprintf("coinbase-%d", height),
                From:      "coinbase",
                To:        minerAddress,
                Amount:    reward,
                Fee:       0,
                Timestamp: time.Now(),
                Nonce:     0,
                Type:      "coinbase",
                Data:      []byte(fmt.Sprintf("Block %d reward", height)),
        }
}

func (pow *BitcoinPoW) SelectValidator(validators []*types.Validator, round int64) (*types.Validator, error) {
        if len(validators) == 0 {
                return &types.Validator{
                        Address:    "bitcoin-miner",
                        PublicKey:  "",
                        Stake:      0,
                        Power:      1.0,
                        LastActive: time.Now(),
                        ShardID:    0,
                        Status:     "active",
                        Reputation: 1.0,
                }, nil
        }

        validatorIndex := round % int64(len(validators))
        selected := validators[validatorIndex]

        pow.logger.LogConsensus("bitcoin_pow", "validator_selected", logrus.Fields{
                "validator":        selected.Address,
                "round":            round,
                "validator_index":  validatorIndex,
                "total_validators": len(validators),
                "timestamp":        time.Now().UTC(),
        })

        return selected, nil
}

func (pow *BitcoinPoW) GetConsensusState() *types.ConsensusState {
        pow.mu.RLock()
        defer pow.mu.RUnlock()

        pow.state.Performance["hash_rate"] = pow.hashRate
        pow.state.Performance["total_hashes"] = float64(pow.totalHashes)
        pow.state.Performance["blocks_found"] = float64(pow.blocksFound)
        pow.state.Performance["difficulty_bits"] = float64(pow.bits)
        pow.state.Performance["uptime"] = time.Since(pow.startTime).Seconds()
        pow.state.Performance["current_height"] = float64(pow.currentHeight)

        return pow.state
}

func (pow *BitcoinPoW) UpdateValidators(validators []*types.Validator) error {
        pow.mu.Lock()
        defer pow.mu.Unlock()

        pow.state.Validators = validators

        pow.logger.LogConsensus("bitcoin_pow", "validators_updated", logrus.Fields{
                "validator_count": len(validators),
                "timestamp":       time.Now().UTC(),
        })

        return nil
}

func (pow *BitcoinPoW) GetAlgorithmName() string {
        return "bitcoin_pow"
}

func (pow *BitcoinPoW) GetMetrics() map[string]interface{} {
        pow.mu.RLock()
        defer pow.mu.RUnlock()

        pow.updateMetrics()
        return pow.metrics
}

func (pow *BitcoinPoW) updateMetrics() {
        uptime := time.Since(pow.startTime)

        pow.metrics["algorithm"] = "bitcoin_pow"
        pow.metrics["difficulty_bits"] = pow.bits
        pow.metrics["difficulty_target"] = pow.difficulty.Text(16)
        pow.metrics["hash_rate"] = pow.hashRate
        pow.metrics["total_hashes"] = pow.totalHashes
        pow.metrics["blocks_found"] = pow.blocksFound
        pow.metrics["uptime_seconds"] = uptime.Seconds()
        pow.metrics["current_height"] = pow.currentHeight
        pow.metrics["extra_nonce"] = pow.extraNonce
        pow.metrics["last_adjustment"] = pow.lastAdjustment
        pow.metrics["avg_time_per_block"] = 0.0

        if pow.blocksFound > 0 {
                pow.metrics["avg_time_per_block"] = uptime.Seconds() / float64(pow.blocksFound)
        }

        pow.metrics["efficiency"] = 0.0
        if pow.totalHashes > 0 {
                pow.metrics["efficiency"] = float64(pow.blocksFound) / float64(pow.totalHashes) * 100
        }

        pow.metrics["timestamp"] = time.Now().UTC()
}

func (pow *BitcoinPoW) Reset() error {
        pow.mu.Lock()
        defer pow.mu.Unlock()

        pow.logger.LogConsensus("bitcoin_pow", "reset", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })

        pow.state.Round = 0
        pow.state.View = 0
        pow.state.Phase = "ready"
        pow.state.Votes = make(map[string]interface{})
        pow.state.LastDecision = time.Now()
        pow.state.Performance = make(map[string]float64)

        pow.totalHashes = 0
        pow.blocksFound = 0
        pow.hashRate = 0
        pow.startTime = time.Now()
        pow.extraNonce = 0
        pow.currentHeight = 0
        pow.blockTimes = make([]time.Time, 0, DifficultyAdjInterval)

        pow.updateMetrics()

        return nil
}

func (pow *BitcoinPoW) StopMining() {
        pow.mu.Lock()
        defer pow.mu.Unlock()

        if pow.miningActive {
                close(pow.stopMining)
                pow.stopMining = make(chan struct{})
                pow.miningActive = false
        }
}

func (pow *BitcoinPoW) GetHashRate() float64 {
        pow.mu.RLock()
        defer pow.mu.RUnlock()
        return pow.hashRate
}

func (pow *BitcoinPoW) GetDifficulty() int {
        pow.mu.RLock()
        defer pow.mu.RUnlock()
        difficultyRatio := new(big.Int).Div(bitsToTarget(0x1d00ffff), pow.difficulty)
        return int(difficultyRatio.Int64())
}

func (pow *BitcoinPoW) GetBits() uint32 {
        pow.mu.RLock()
        defer pow.mu.RUnlock()
        return pow.bits
}

func (pow *BitcoinPoW) GetTarget() *big.Int {
        pow.mu.RLock()
        defer pow.mu.RUnlock()
        return new(big.Int).Set(pow.difficulty)
}

func (pow *BitcoinPoW) AdjustDifficulty(avgBlockTime float64, targetBlockTime float64) {
        pow.adjustDifficultyBitcoin()
}
