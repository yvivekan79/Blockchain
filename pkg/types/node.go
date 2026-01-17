package types

import "time"

// NodeRole defines the role of a node in the network
type NodeRole string

const (
	RoleBootstrap NodeRole = "bootstrap"
	RoleValidator NodeRole = "validator"
	RoleObserver  NodeRole = "observer"
)

// ConsensusAlgorithm defines supported consensus algorithms
type ConsensusAlgorithm string

const (
	AlgorithmPoW   ConsensusAlgorithm = "pow"
	AlgorithmPoS   ConsensusAlgorithm = "pos"
	AlgorithmPBFT  ConsensusAlgorithm = "pbft"
	AlgorithmPPBFT ConsensusAlgorithm = "ppbft"
	AlgorithmLSCC  ConsensusAlgorithm = "lscc"
)

// NodeInfo contains information about a network node
type NodeInfo struct {
	ID                 string             `json:"id" yaml:"id"`
	Name               string             `json:"name" yaml:"name"`
	Description        string             `json:"description" yaml:"description"`
	ConsensusAlgorithm ConsensusAlgorithm `json:"consensus_algorithm" yaml:"consensus_algorithm"`
	Role               NodeRole           `json:"role" yaml:"role"`
	ExternalIP         string             `json:"external_ip" yaml:"external_ip"`
	Region             string             `json:"region" yaml:"region"`
	StartTime          time.Time          `json:"start_time"`
	LastSeen           time.Time          `json:"last_seen"`
	Version            string             `json:"version"`
}

// NetworkPeer represents a peer in the network
type NetworkPeer struct {
	NodeInfo
	Address     string        `json:"address"`
	Port        int           `json:"port"`
	Connected   bool          `json:"connected"`
	Latency     time.Duration `json:"latency"`
	MessagesSent int64        `json:"messages_sent"`
	MessagesReceived int64    `json:"messages_received"`
	LastPing    time.Time     `json:"last_ping"`
}

// BootstrapConfig contains bootstrap node configuration
type BootstrapConfig struct {
	Enabled          bool   `json:"enabled" yaml:"enabled"`
	AdvertiseAddress string `json:"advertise_address" yaml:"advertise_address"`
}

// CrossAlgorithmMessage represents communication between different consensus algorithms
type CrossAlgorithmMessage struct {
	FromAlgorithm ConsensusAlgorithm `json:"from_algorithm"`
	ToAlgorithm   ConsensusAlgorithm `json:"to_algorithm"`
	MessageType   string             `json:"message_type"`
	Payload       interface{}        `json:"payload"`
	Timestamp     time.Time          `json:"timestamp"`
	MessageID     string             `json:"message_id"`
}

// AlgorithmPeerInfo contains information about peers running specific algorithms
type AlgorithmPeerInfo struct {
	Algorithm  ConsensusAlgorithm `json:"algorithm"`
	PeerCount  int                `json:"peer_count"`
	Peers      []NetworkPeer      `json:"peers"`
	LastUpdate time.Time          `json:"last_update"`
}