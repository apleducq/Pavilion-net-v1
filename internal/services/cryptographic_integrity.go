package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// CryptographicIntegrityService handles cryptographic integrity operations
type CryptographicIntegrityService struct {
	config *config.Config
}

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	Hash     string      `json:"hash"`
	Left     *MerkleNode `json:"left,omitempty"`
	Right    *MerkleNode `json:"right,omitempty"`
	IsLeaf   bool        `json:"is_leaf"`
	Data     string      `json:"data,omitempty"`
	Position int         `json:"position,omitempty"`
}

// MerkleProof represents a Merkle proof for verification
type MerkleProof struct {
	RootHash   string   `json:"root_hash"`
	LeafHash   string   `json:"leaf_hash"`
	ProofPath  []string `json:"proof_path"`
	ProofIndex []int    `json:"proof_index"`
	TreeHeight int      `json:"tree_height"`
	LeafCount  int      `json:"leaf_count"`
	Timestamp  string   `json:"timestamp"`
}

// HashChain represents a hash chain for audit trail
type HashChain struct {
	PreviousHash string `json:"previous_hash"`
	CurrentHash  string `json:"current_hash"`
	Timestamp    string `json:"timestamp"`
	EntryID      string `json:"entry_id"`
	Sequence     int64  `json:"sequence"`
}

// NewCryptographicIntegrityService creates a new cryptographic integrity service
func NewCryptographicIntegrityService(cfg *config.Config) *CryptographicIntegrityService {
	return &CryptographicIntegrityService{
		config: cfg,
	}
}

// GenerateHash generates a SHA-256 hash for the given data
func (s *CryptographicIntegrityService) GenerateHash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GenerateEntryHash generates a hash for an audit entry
func (s *CryptographicIntegrityService) GenerateEntryHash(entry *models.AuditEntry) string {
	// Create a deterministic string representation of the entry
	entryData := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s",
		entry.Timestamp,
		entry.RequestID,
		entry.RPID,
		entry.ClaimType,
		entry.PrivacyHash,
		entry.Status,
		entry.PolicyDecision,
	)
	return s.GenerateHash(entryData)
}

// BuildMerkleTree constructs a Merkle tree from audit entries
func (s *CryptographicIntegrityService) BuildMerkleTree(entries []*models.AuditEntry) *MerkleNode {
	if len(entries) == 0 {
		return nil
	}

	// Generate leaf nodes
	var leaves []*MerkleNode
	for i, entry := range entries {
		hash := s.GenerateEntryHash(entry)
		leaves = append(leaves, &MerkleNode{
			Hash:     hash,
			IsLeaf:   true,
			Data:     fmt.Sprintf("%s:%s", entry.RequestID, entry.Timestamp),
			Position: i,
		})
	}

	// Build the tree from leaves to root
	return s.buildTreeFromLeaves(leaves)
}

// buildTreeFromLeaves recursively builds the Merkle tree
func (s *CryptographicIntegrityService) buildTreeFromLeaves(nodes []*MerkleNode) *MerkleNode {
	if len(nodes) == 1 {
		return nodes[0]
	}

	var parents []*MerkleNode
	for i := 0; i < len(nodes); i += 2 {
		left := nodes[i]
		var right *MerkleNode

		if i+1 < len(nodes) {
			right = nodes[i+1]
		} else {
			// Duplicate the left node if there's no right sibling
			right = &MerkleNode{
				Hash:   left.Hash,
				IsLeaf: left.IsLeaf,
				Data:   left.Data,
			}
		}

		// Create parent node
		parentHash := s.GenerateHash(left.Hash + right.Hash)
		parent := &MerkleNode{
			Hash:   parentHash,
			Left:   left,
			Right:  right,
			IsLeaf: false,
		}
		parents = append(parents, parent)
	}

	return s.buildTreeFromLeaves(parents)
}

// GenerateMerkleProof generates a Merkle proof for a specific entry
func (s *CryptographicIntegrityService) GenerateMerkleProof(entries []*models.AuditEntry, targetEntryID string) (*MerkleProof, error) {
	// Find the target entry
	var targetEntry *models.AuditEntry
	var targetIndex int
	for i, entry := range entries {
		if entry.RequestID == targetEntryID {
			targetEntry = entry
			targetIndex = i
			break
		}
	}

	if targetEntry == nil {
		return nil, fmt.Errorf("target entry not found: %s", targetEntryID)
	}

	// Build the Merkle tree
	root := s.BuildMerkleTree(entries)
	if root == nil {
		return nil, fmt.Errorf("failed to build Merkle tree")
	}

	// Generate proof path
	proofPath, proofIndex := s.generateProofPath(root, targetIndex, len(entries))

	return &MerkleProof{
		RootHash:   root.Hash,
		LeafHash:   s.GenerateEntryHash(targetEntry),
		ProofPath:  proofPath,
		ProofIndex: proofIndex,
		TreeHeight: s.calculateTreeHeight(len(entries)),
		LeafCount:  len(entries),
		Timestamp:  time.Now().Format(time.RFC3339),
	}, nil
}

// generateProofPath generates the proof path for a leaf node
func (s *CryptographicIntegrityService) generateProofPath(node *MerkleNode, targetIndex, totalLeaves int) ([]string, []int) {
	if node == nil || node.IsLeaf {
		return []string{}, []int{}
	}

	var proofPath []string
	var proofIndex []int

	// Calculate which half the target is in
	mid := (totalLeaves + 1) / 2
	if targetIndex < mid {
		// Target is in left subtree
		if node.Right != nil {
			proofPath = append(proofPath, node.Right.Hash)
			proofIndex = append(proofIndex, 1) // Right sibling
		}
		subProof, subIndex := s.generateProofPath(node.Left, targetIndex, mid)
		proofPath = append(proofPath, subProof...)
		proofIndex = append(proofIndex, subIndex...)
	} else {
		// Target is in right subtree
		if node.Left != nil {
			proofPath = append(proofPath, node.Left.Hash)
			proofIndex = append(proofIndex, 0) // Left sibling
		}
		subProof, subIndex := s.generateProofPath(node.Right, targetIndex-mid, totalLeaves-mid)
		proofPath = append(proofPath, subProof...)
		proofIndex = append(proofIndex, subIndex...)
	}

	return proofPath, proofIndex
}

// calculateTreeHeight calculates the height of the Merkle tree
func (s *CryptographicIntegrityService) calculateTreeHeight(leafCount int) int {
	if leafCount == 0 {
		return 0
	}

	height := 0
	for leafCount > 1 {
		leafCount = (leafCount + 1) / 2
		height++
	}
	return height
}

// VerifyMerkleProof verifies a Merkle proof
func (s *CryptographicIntegrityService) VerifyMerkleProof(proof *MerkleProof, leafData string) bool {
	if proof == nil || len(proof.ProofPath) == 0 {
		return false
	}

	// Start with the leaf hash
	currentHash := proof.LeafHash

	// Reconstruct the path to the root
	for i, siblingHash := range proof.ProofPath {
		if proof.ProofIndex[i] == 0 {
			// Left sibling, hash as left + right
			currentHash = s.GenerateHash(siblingHash + currentHash)
		} else {
			// Right sibling, hash as left + right
			currentHash = s.GenerateHash(currentHash + siblingHash)
		}
	}

	// Compare with root hash
	return currentHash == proof.RootHash
}

// CreateHashChain creates a hash chain entry
func (s *CryptographicIntegrityService) CreateHashChain(previousHash, entryID string, sequence int64) *HashChain {
	// Create current hash from previous hash and entry data
	entryData := fmt.Sprintf("%s:%s:%d:%s", previousHash, entryID, sequence, time.Now().Format(time.RFC3339))
	currentHash := s.GenerateHash(entryData)

	return &HashChain{
		PreviousHash: previousHash,
		CurrentHash:  currentHash,
		Timestamp:    time.Now().Format(time.RFC3339),
		EntryID:      entryID,
		Sequence:     sequence,
	}
}

// VerifyHashChain verifies a hash chain
func (s *CryptographicIntegrityService) VerifyHashChain(chain *HashChain) bool {
	if chain == nil {
		return false
	}

	// Recreate the current hash
	entryData := fmt.Sprintf("%s:%s:%d:%s",
		chain.PreviousHash, chain.EntryID, chain.Sequence, chain.Timestamp)
	expectedHash := s.GenerateHash(entryData)

	return expectedHash == chain.CurrentHash
}

// GenerateIntegrityHash generates an integrity hash for a batch of entries
func (s *CryptographicIntegrityService) GenerateIntegrityHash(entries []*models.AuditEntry) string {
	if len(entries) == 0 {
		return ""
	}

	// Sort entries by timestamp for deterministic ordering
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp < entries[j].Timestamp
	})

	// Create a combined hash of all entries
	var hashes []string
	for _, entry := range entries {
		hashes = append(hashes, s.GenerateEntryHash(entry))
	}

	// Combine all hashes
	combinedData := strings.Join(hashes, ":")
	return s.GenerateHash(combinedData)
}

// ValidateAuditIntegrity validates the integrity of audit entries
func (s *CryptographicIntegrityService) ValidateAuditIntegrity(entries []*models.AuditEntry) error {
	if len(entries) == 0 {
		return fmt.Errorf("no entries to validate")
	}

	// Check for duplicate request IDs
	requestIDs := make(map[string]bool)
	for _, entry := range entries {
		if requestIDs[entry.RequestID] {
			return fmt.Errorf("duplicate request ID found: %s", entry.RequestID)
		}
		requestIDs[entry.RequestID] = true
	}

	// Validate each entry
	for _, entry := range entries {
		if err := s.validateAuditEntry(entry); err != nil {
			return fmt.Errorf("invalid audit entry %s: %w", entry.RequestID, err)
		}
	}

	return nil
}

// validateAuditEntry validates a single audit entry
func (s *CryptographicIntegrityService) validateAuditEntry(entry *models.AuditEntry) error {
	if entry.RequestID == "" {
		return fmt.Errorf("request ID is empty")
	}

	if entry.Timestamp == "" {
		return fmt.Errorf("timestamp is empty")
	}

	if entry.PrivacyHash == "" {
		return fmt.Errorf("privacy hash is empty")
	}

	if entry.MerkleProof == "" {
		return fmt.Errorf("merkle proof is empty")
	}

	// Validate timestamp format
	_, err := time.Parse(time.RFC3339, entry.Timestamp)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}

	return nil
}

// HealthCheck checks if the cryptographic integrity service is healthy
func (s *CryptographicIntegrityService) HealthCheck(ctx context.Context) error {
	// Test hash generation
	testData := "test_integrity_check"
	hash := s.GenerateHash(testData)
	if hash == "" {
		return fmt.Errorf("hash generation failed")
	}

	// Test Merkle tree construction with sample data
	sampleEntries := []*models.AuditEntry{
		{
			RequestID:   "test1",
			Timestamp:   time.Now().Format(time.RFC3339),
			PrivacyHash: "test_hash_1",
			MerkleProof: "test_proof_1",
		},
		{
			RequestID:   "test2",
			Timestamp:   time.Now().Format(time.RFC3339),
			PrivacyHash: "test_hash_2",
			MerkleProof: "test_proof_2",
		},
	}

	root := s.BuildMerkleTree(sampleEntries)
	if root == nil {
		return fmt.Errorf("merkle tree construction failed")
	}

	return nil
}
