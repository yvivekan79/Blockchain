package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"lscc-blockchain/pkg/types"
)

// MerkleTree represents a Merkle tree
type MerkleTree struct {
	Root  *MerkleNode
	Leafs []*MerkleNode
}

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
	Hash  string
}

// NewMerkleTree creates a new Merkle tree from transactions
func NewMerkleTree(transactions []*types.Transaction) *MerkleTree {
	var leafs []*MerkleNode
	
	// Create leaf nodes from transactions
	for _, tx := range transactions {
		hash := sha256.Sum256([]byte(tx.ID))
		leaf := &MerkleNode{
			Data: []byte(tx.ID),
			Hash: hex.EncodeToString(hash[:]),
		}
		leafs = append(leafs, leaf)
	}
	
	// Handle empty transaction list
	if len(leafs) == 0 {
		hash := sha256.Sum256([]byte(""))
		leaf := &MerkleNode{
			Data: []byte(""),
			Hash: hex.EncodeToString(hash[:]),
		}
		leafs = append(leafs, leaf)
	}
	
	// Build the tree
	root := buildTree(leafs)
	
	return &MerkleTree{
		Root:  root,
		Leafs: leafs,
	}
}

// buildTree recursively builds the Merkle tree
func buildTree(nodes []*MerkleNode) *MerkleNode {
	if len(nodes) == 1 {
		return nodes[0]
	}
	
	var newLevel []*MerkleNode
	
	for i := 0; i < len(nodes); i += 2 {
		var left, right *MerkleNode
		left = nodes[i]
		
		if i+1 < len(nodes) {
			right = nodes[i+1]
		} else {
			// Duplicate the last node if odd number of nodes
			right = nodes[i]
		}
		
		// Create parent node
		parent := &MerkleNode{
			Left:  left,
			Right: right,
		}
		
		// Calculate parent hash
		combinedHash := left.Hash + right.Hash
		hash := sha256.Sum256([]byte(combinedHash))
		parent.Hash = hex.EncodeToString(hash[:])
		
		newLevel = append(newLevel, parent)
	}
	
	return buildTree(newLevel)
}

// GetRootHash returns the root hash of the Merkle tree
func (mt *MerkleTree) GetRootHash() string {
	if mt.Root == nil {
		return ""
	}
	return mt.Root.Hash
}

// GenerateMerkleProof generates a Merkle proof for a transaction
func (mt *MerkleTree) GenerateMerkleProof(txID string) ([]MerkleProofElement, error) {
	var proof []MerkleProofElement
	
	// Find the leaf node for the transaction
	var targetLeaf *MerkleNode
	for _, leaf := range mt.Leafs {
		if string(leaf.Data) == txID {
			targetLeaf = leaf
			break
		}
	}
	
	if targetLeaf == nil {
		return nil, nil
	}
	
	// Generate proof by traversing up the tree
	proof = generateProofPath(mt.Root, targetLeaf, proof)
	
	return proof, nil
}

// MerkleProofElement represents an element in a Merkle proof
type MerkleProofElement struct {
	Hash      string `json:"hash"`
	Direction string `json:"direction"` // "left" or "right"
}

// generateProofPath recursively generates the proof path
func generateProofPath(node *MerkleNode, target *MerkleNode, proof []MerkleProofElement) []MerkleProofElement {
	if node == nil || node == target {
		return proof
	}
	
	if node.Left == target {
		// Target is left child, add right sibling to proof
		if node.Right != nil {
			proof = append(proof, MerkleProofElement{
				Hash:      node.Right.Hash,
				Direction: "right",
			})
		}
		return proof
	}
	
	if node.Right == target {
		// Target is right child, add left sibling to proof
		if node.Left != nil {
			proof = append(proof, MerkleProofElement{
				Hash:      node.Left.Hash,
				Direction: "left",
			})
		}
		return proof
	}
	
	// Check if target is in left subtree
	if containsNode(node.Left, target) {
		if node.Right != nil {
			proof = append(proof, MerkleProofElement{
				Hash:      node.Right.Hash,
				Direction: "right",
			})
		}
		return generateProofPath(node.Left, target, proof)
	}
	
	// Check if target is in right subtree
	if containsNode(node.Right, target) {
		if node.Left != nil {
			proof = append(proof, MerkleProofElement{
				Hash:      node.Left.Hash,
				Direction: "left",
			})
		}
		return generateProofPath(node.Right, target, proof)
	}
	
	return proof
}

// containsNode checks if a subtree contains a target node
func containsNode(root *MerkleNode, target *MerkleNode) bool {
	if root == nil {
		return false
	}
	
	if root == target {
		return true
	}
	
	return containsNode(root.Left, target) || containsNode(root.Right, target)
}

// VerifyMerkleProof verifies a Merkle proof
func VerifyMerkleProof(rootHash string, txID string, proof []MerkleProofElement) bool {
	// Start with the transaction hash
	txHash := sha256.Sum256([]byte(txID))
	currentHash := hex.EncodeToString(txHash[:])
	
	// Apply each proof element
	for _, element := range proof {
		var combinedHash string
		
		if element.Direction == "left" {
			combinedHash = element.Hash + currentHash
		} else {
			combinedHash = currentHash + element.Hash
		}
		
		hash := sha256.Sum256([]byte(combinedHash))
		currentHash = hex.EncodeToString(hash[:])
	}
	
	return currentHash == rootHash
}

// GetLeafCount returns the number of leaf nodes
func (mt *MerkleTree) GetLeafCount() int {
	return len(mt.Leafs)
}

// GetDepth returns the depth of the Merkle tree
func (mt *MerkleTree) GetDepth() int {
	if mt.Root == nil {
		return 0
	}
	return getNodeDepth(mt.Root)
}

// getNodeDepth recursively calculates the depth of a node
func getNodeDepth(node *MerkleNode) int {
	if node == nil {
		return 0
	}
	
	if node.Left == nil && node.Right == nil {
		return 1
	}
	
	leftDepth := getNodeDepth(node.Left)
	rightDepth := getNodeDepth(node.Right)
	
	if leftDepth > rightDepth {
		return leftDepth + 1
	}
	return rightDepth + 1
}

// Print prints the Merkle tree structure (for debugging)
func (mt *MerkleTree) Print() {
	if mt.Root == nil {
		return
	}
	printNode(mt.Root, 0)
}

// printNode recursively prints a node and its children
func printNode(node *MerkleNode, depth int) {
	if node == nil {
		return
	}
	
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	
	if len(node.Data) > 0 {
		// Leaf node
		println(indent + "LEAF: " + string(node.Data) + " -> " + node.Hash[:8] + "...")
	} else {
		// Internal node
		println(indent + "NODE: " + node.Hash[:8] + "...")
	}
	
	printNode(node.Left, depth+1)
	printNode(node.Right, depth+1)
}

// CreateMerkleRootFromHashes creates a Merkle root from a list of hashes
func CreateMerkleRootFromHashes(hashes []string) string {
	if len(hashes) == 0 {
		hash := sha256.Sum256([]byte(""))
		return hex.EncodeToString(hash[:])
	}
	
	if len(hashes) == 1 {
		return hashes[0]
	}
	
	var newLevel []string
	
	for i := 0; i < len(hashes); i += 2 {
		var left, right string
		left = hashes[i]
		
		if i+1 < len(hashes) {
			right = hashes[i+1]
		} else {
			right = hashes[i]
		}
		
		combinedHash := left + right
		hash := sha256.Sum256([]byte(combinedHash))
		newLevel = append(newLevel, hex.EncodeToString(hash[:]))
	}
	
	return CreateMerkleRootFromHashes(newLevel)
}
