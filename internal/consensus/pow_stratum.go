package consensus

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"lscc-blockchain/internal/utils"
	"lscc-blockchain/pkg/types"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type StratumJob struct {
	JobID         string
	PrevBlockHash string
	CoinbaseData1 string
	CoinbaseData2 string
	MerkleBranch  []string
	Version       string
	Bits          string
	Timestamp     string
	CleanJobs     bool
	Target        *big.Int
	BlockHeight   int64
	CreatedAt     time.Time
}

type StratumWorker struct {
	ID            string
	Name          string
	Address       string
	Conn          net.Conn
	Authorized    bool
	Subscribed    bool
	ExtraNonce1   string
	ExtraNonce2   string
	Difficulty    float64
	SharesValid   int64
	SharesInvalid int64
	LastShare     time.Time
	HashRate      float64
}

type StratumServer struct {
	logger         *utils.Logger
	pow            *BitcoinPoW
	listener       net.Listener
	workers        map[string]*StratumWorker
	currentJob     *StratumJob
	jobCounter     uint64
	mu             sync.RWMutex
	running        bool
	port           int
	extraNonce1Len int
	extraNonce2Len int
	difficulty     float64
	submitChan     chan *ShareSubmit
	blockCallback  func(*types.Block) error
}

type ShareSubmit struct {
	WorkerID    string
	JobID       string
	ExtraNonce2 string
	Nonce       string
	Timestamp   string
}

type StratumRequest struct {
	ID     interface{}   `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

type StratumResponse struct {
	ID     interface{}   `json:"id"`
	Result interface{}   `json:"result,omitempty"`
	Error  []interface{} `json:"error,omitempty"`
}

type StratumNotification struct {
	ID     interface{}   `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

func NewStratumServer(pow *BitcoinPoW, logger *utils.Logger, port int) *StratumServer {
	return &StratumServer{
		logger:         logger,
		pow:            pow,
		workers:        make(map[string]*StratumWorker),
		port:           port,
		extraNonce1Len: 4,
		extraNonce2Len: 4,
		difficulty:     1.0,
		submitChan:     make(chan *ShareSubmit, 100),
		running:        false,
	}
}

func (s *StratumServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start stratum server: %w", err)
	}

	s.listener = listener
	s.running = true

	s.logger.LogConsensus("stratum", "server_started", logrus.Fields{
		"port":      s.port,
		"timestamp": time.Now().UTC(),
	})

	go s.acceptConnections()
	go s.processShares()

	return nil
}

func (s *StratumServer) Stop() error {
	s.running = false
	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	for _, worker := range s.workers {
		if worker.Conn != nil {
			worker.Conn.Close()
		}
	}
	s.workers = make(map[string]*StratumWorker)
	s.mu.Unlock()

	s.logger.LogConsensus("stratum", "server_stopped", logrus.Fields{
		"timestamp": time.Now().UTC(),
	})

	return nil
}

func (s *StratumServer) acceptConnections() {
	for s.running {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.running {
				s.logger.LogError("stratum", "accept_error", err, nil)
			}
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *StratumServer) handleConnection(conn net.Conn) {
	workerID := s.generateWorkerID()
	extraNonce1 := s.generateExtraNonce1()

	worker := &StratumWorker{
		ID:          workerID,
		Conn:        conn,
		ExtraNonce1: extraNonce1,
		Difficulty:  s.difficulty,
		Address:     conn.RemoteAddr().String(),
	}

	s.mu.Lock()
	s.workers[workerID] = worker
	s.mu.Unlock()

	s.logger.LogConsensus("stratum", "worker_connected", logrus.Fields{
		"worker_id": workerID,
		"address":   conn.RemoteAddr().String(),
		"timestamp": time.Now().UTC(),
	})

	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.workers, workerID)
		s.mu.Unlock()

		s.logger.LogConsensus("stratum", "worker_disconnected", logrus.Fields{
			"worker_id": workerID,
			"timestamp": time.Now().UTC(),
		})
	}()

	decoder := json.NewDecoder(conn)

	for s.running {
		var req StratumRequest
		if err := decoder.Decode(&req); err != nil {
			break
		}

		s.handleRequest(worker, &req)
	}
}

func (s *StratumServer) handleRequest(worker *StratumWorker, req *StratumRequest) {
	switch req.Method {
	case "mining.subscribe":
		s.handleSubscribe(worker, req)
	case "mining.authorize":
		s.handleAuthorize(worker, req)
	case "mining.submit":
		s.handleSubmit(worker, req)
	case "mining.extranonce.subscribe":
		s.handleExtranonceSubscribe(worker, req)
	default:
		s.sendError(worker, req.ID, 20, "Unknown method", nil)
	}
}

func (s *StratumServer) handleSubscribe(worker *StratumWorker, req *StratumRequest) {
	worker.Subscribed = true

	result := []interface{}{
		[][]interface{}{
			{"mining.set_difficulty", "subscription_id_1"},
			{"mining.notify", "subscription_id_2"},
		},
		worker.ExtraNonce1,
		s.extraNonce2Len,
	}

	s.sendResponse(worker, req.ID, result, nil)

	s.sendDifficulty(worker)

	if s.currentJob != nil {
		s.sendJob(worker)
	}

	s.logger.LogConsensus("stratum", "worker_subscribed", logrus.Fields{
		"worker_id":    worker.ID,
		"extra_nonce1": worker.ExtraNonce1,
		"timestamp":    time.Now().UTC(),
	})
}

func (s *StratumServer) handleAuthorize(worker *StratumWorker, req *StratumRequest) {
	if len(req.Params) >= 1 {
		if name, ok := req.Params[0].(string); ok {
			worker.Name = name
		}
	}

	worker.Authorized = true
	s.sendResponse(worker, req.ID, true, nil)

	s.logger.LogConsensus("stratum", "worker_authorized", logrus.Fields{
		"worker_id":   worker.ID,
		"worker_name": worker.Name,
		"timestamp":   time.Now().UTC(),
	})
}

func (s *StratumServer) handleSubmit(worker *StratumWorker, req *StratumRequest) {
	if !worker.Authorized {
		s.sendError(worker, req.ID, 24, "Unauthorized", nil)
		return
	}

	if len(req.Params) < 5 {
		s.sendError(worker, req.ID, 21, "Invalid parameters", nil)
		worker.SharesInvalid++
		return
	}

	submit := &ShareSubmit{
		WorkerID: worker.ID,
	}

	if jobID, ok := req.Params[1].(string); ok {
		submit.JobID = jobID
	}
	if en2, ok := req.Params[2].(string); ok {
		submit.ExtraNonce2 = en2
	}
	if ts, ok := req.Params[3].(string); ok {
		submit.Timestamp = ts
	}
	if nonce, ok := req.Params[4].(string); ok {
		submit.Nonce = nonce
	}

	s.submitChan <- submit
	s.sendResponse(worker, req.ID, true, nil)
}

func (s *StratumServer) handleExtranonceSubscribe(worker *StratumWorker, req *StratumRequest) {
	s.sendResponse(worker, req.ID, true, nil)
}

func (s *StratumServer) processShares() {
	for submit := range s.submitChan {
		s.validateShare(submit)
	}
}

func (s *StratumServer) validateShare(submit *ShareSubmit) bool {
	s.mu.RLock()
	worker, exists := s.workers[submit.WorkerID]
	s.mu.RUnlock()

	if !exists {
		return false
	}

	if s.currentJob == nil || submit.JobID != s.currentJob.JobID {
		worker.SharesInvalid++
		return false
	}

	worker.SharesValid++
	worker.LastShare = time.Now()

	s.logger.LogConsensus("stratum", "share_accepted", logrus.Fields{
		"worker_id":    worker.ID,
		"job_id":       submit.JobID,
		"nonce":        submit.Nonce,
		"valid_shares": worker.SharesValid,
		"timestamp":    time.Now().UTC(),
	})

	return true
}

func (s *StratumServer) CreateJob(block *types.Block) *StratumJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobCounter++

	job := &StratumJob{
		JobID:         fmt.Sprintf("%08x", s.jobCounter),
		PrevBlockHash: block.PreviousHash,
		MerkleBranch:  []string{},
		Version:       "00000001",
		Bits:          fmt.Sprintf("%08x", s.pow.GetBits()),
		Timestamp:     fmt.Sprintf("%08x", uint32(time.Now().Unix())),
		CleanJobs:     true,
		Target:        s.pow.GetTarget(),
		BlockHeight:   block.Index,
		CreatedAt:     time.Now(),
	}

	s.currentJob = job

	s.broadcastJob()

	s.logger.LogConsensus("stratum", "job_created", logrus.Fields{
		"job_id":       job.JobID,
		"block_height": job.BlockHeight,
		"bits":         job.Bits,
		"timestamp":    time.Now().UTC(),
	})

	return job
}

func (s *StratumServer) broadcastJob() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, worker := range s.workers {
		if worker.Subscribed {
			s.sendJob(worker)
		}
	}
}

func (s *StratumServer) sendJob(worker *StratumWorker) {
	if s.currentJob == nil {
		return
	}

	job := s.currentJob

	notification := StratumNotification{
		ID:     nil,
		Method: "mining.notify",
		Params: []interface{}{
			job.JobID,
			job.PrevBlockHash,
			job.CoinbaseData1,
			job.CoinbaseData2,
			job.MerkleBranch,
			job.Version,
			job.Bits,
			job.Timestamp,
			job.CleanJobs,
		},
	}

	s.sendNotification(worker, &notification)
}

func (s *StratumServer) sendDifficulty(worker *StratumWorker) {
	notification := StratumNotification{
		ID:     nil,
		Method: "mining.set_difficulty",
		Params: []interface{}{worker.Difficulty},
	}

	s.sendNotification(worker, &notification)
}

func (s *StratumServer) sendResponse(worker *StratumWorker, id interface{}, result interface{}, error []interface{}) {
	resp := StratumResponse{
		ID:     id,
		Result: result,
		Error:  error,
	}

	data, _ := json.Marshal(resp)
	data = append(data, '\n')
	worker.Conn.Write(data)
}

func (s *StratumServer) sendError(worker *StratumWorker, id interface{}, code int, message string, data interface{}) {
	s.sendResponse(worker, id, nil, []interface{}{code, message, data})
}

func (s *StratumServer) sendNotification(worker *StratumWorker, notification *StratumNotification) {
	data, _ := json.Marshal(notification)
	data = append(data, '\n')
	worker.Conn.Write(data)
}

func (s *StratumServer) generateWorkerID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *StratumServer) generateExtraNonce1() string {
	bytes := make([]byte, s.extraNonce1Len)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *StratumServer) GetWorkerStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["worker_count"] = len(s.workers)

	var totalShares int64
	var totalHashRate float64
	workers := make([]map[string]interface{}, 0)

	for _, worker := range s.workers {
		totalShares += worker.SharesValid
		totalHashRate += worker.HashRate

		workers = append(workers, map[string]interface{}{
			"id":             worker.ID,
			"name":           worker.Name,
			"address":        worker.Address,
			"shares_valid":   worker.SharesValid,
			"shares_invalid": worker.SharesInvalid,
			"difficulty":     worker.Difficulty,
			"last_share":     worker.LastShare,
		})
	}

	stats["total_shares"] = totalShares
	stats["total_hash_rate"] = totalHashRate
	stats["workers"] = workers

	if s.currentJob != nil {
		stats["current_job"] = map[string]interface{}{
			"job_id":       s.currentJob.JobID,
			"block_height": s.currentJob.BlockHeight,
			"bits":         s.currentJob.Bits,
			"created_at":   s.currentJob.CreatedAt,
		}
	}

	return stats
}

func (s *StratumServer) SetDifficulty(difficulty float64) {
	s.mu.Lock()
	s.difficulty = difficulty
	s.mu.Unlock()

	s.mu.RLock()
	for _, worker := range s.workers {
		worker.Difficulty = difficulty
		if worker.Subscribed {
			s.sendDifficulty(worker)
		}
	}
	s.mu.RUnlock()
}

func (s *StratumServer) SetBlockCallback(callback func(*types.Block) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blockCallback = callback
}
