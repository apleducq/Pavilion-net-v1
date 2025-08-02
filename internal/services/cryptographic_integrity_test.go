package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestNewCryptographicIntegrityService(t *testing.T) {
	cfg := &config.Config{}
	service := NewCryptographicIntegrityService(cfg)

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.config != cfg {
		t.Error("Expected config to be set")
	}
}

func TestGenerateHash(t *testing.T) {
	service := NewCryptographicIntegrityService(&config.Config{})

	// Test basic hash generation
	data := "test_data"
	hash := service.GenerateHash(data)

	if hash == "" {
		t.Fatal("Expected hash to be generated")
	}

	// Test deterministic hashing
	hash2 := service.GenerateHash(data)
	if hash != hash2 {
		t.Error("Expected deterministic hashing")
	}

	// Test different data produces different hash
	hash3 := service.GenerateHash("different_data")
	if hash == hash3 {
		t.Error("Expected different data to produce different hash")
	}
}

func TestGenerateEntryHash(t *testing.T) {
	service := NewCryptographicIntegrityService(&config.Config{})

	entry := &models.AuditEntry{
		Timestamp:      "2023-01-01T00:00:00Z",
		RequestID:      "req-123",
		RPID:           "rp-001",
		ClaimType:      "student_verification",
		PrivacyHash:    "privacy_hash_123",
		Status:         "SUCCESS",
		PolicyDecision: "ALLOW",
	}

	hash := service.GenerateEntryHash(entry)

	if hash == "" {
		t.Fatal("Expected entry hash to be generated")
	}

	// Test deterministic hashing
	hash2 := service.GenerateEntryHash(entry)
	if hash != hash2 {
		t.Error("Expected deterministic entry hashing")
	}
}

func TestBuildMerkleTree(t *testing.T) {
	service := NewCryptographicIntegrityService(&config.Config{})

	// Test empty entries
	root := service.BuildMerkleTree([]*models.AuditEntry{})
	if root != nil {
		t.Error("Expected nil root for empty entries")
	}

	// Test single entry
	entries := []*models.AuditEntry{
		{
			Timestamp:      "2023-01-01T00:00:00Z",
			RequestID:      "req-1",
			RPID:           "rp-001",
			ClaimType:      "student_verification",
			PrivacyHash:    "hash-1",
			Status:         "SUCCESS",
			PolicyDecision: "ALLOW",
		},
	}

	root = service.BuildMerkleTree(entries)
	if root == nil {
		t.Fatal("Expected root to be created for single entry")
	}

	if !root.IsLeaf {
		t.Error("Expected single entry to be a leaf")
	}

	// Test multiple entries
	entries = append(entries, &models.AuditEntry{
		Timestamp:      "2023-01-01T00:00:01Z",
		RequestID:      "req-2",
		RPID:           "rp-001",
		ClaimType:      "student_verification",
		PrivacyHash:    "hash-2",
		Status:         "SUCCESS",
		PolicyDecision: "ALLOW",
	})

	root = service.BuildMerkleTree(entries)
	if root == nil {
		t.Fatal("Expected root to be created for multiple entries")
	}

	if root.IsLeaf {
		t.Error("Expected root to be internal node for multiple entries")
	}

	if root.Left == nil || root.Right == nil {
		t.Error("Expected root to have children")
	}
}

func TestGenerateMerkleProof(t *testing.T) {
	service := NewCryptographicIntegrityService(&config.Config{})

	entries := []*models.AuditEntry{
		{
			Timestamp:      "2023-01-01T00:00:00Z",
			RequestID:      "req-1",
			RPID:           "rp-001",
			ClaimType:      "student_verification",
			PrivacyHash:    "hash-1",
			Status:         "SUCCESS",
			PolicyDecision: "ALLOW",
		},
		{
			Timestamp:      "2023-01-01T00:00:01Z",
			RequestID:      "req-2",
			RPID:           "rp-001",
			ClaimType:      "student_verification",
			PrivacyHash:    "hash-2",
			Status:         "SUCCESS",
			PolicyDecision: "ALLOW",
		},
	}

	// Test proof generation for existing entry
	proof, err := service.GenerateMerkleProof(entries, "req-1")
	if err != nil {
		t.Fatalf("Expected no error: %v", err)
	}

	if proof == nil {
		t.Fatal("Expected proof to be generated")
	}

	if proof.RootHash == "" {
		t.Error("Expected root hash to be set")
	}

	if proof.LeafHash == "" {
		t.Error("Expected leaf hash to be set")
	}

	if len(proof.ProofPath) == 0 {
		t.Error("Expected proof path to be generated")
	}

	// Test proof generation for non-existent entry
	_, err = service.GenerateMerkleProof(entries, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent entry")
	}
}

func TestVerifyMerkleProof(t *testing.T) {
	service := NewCryptographicIntegrityService(&config.Config{})

	entries := []*models.AuditEntry{
		{
			Timestamp:      "2023-01-01T00:00:00Z",
			RequestID:      "req-1",
			RPID:           "rp-001",
			ClaimType:      "student_verification",
			PrivacyHash:    "hash-1",
			Status:         "SUCCESS",
			PolicyDecision: "ALLOW",
		},
		{
			Timestamp:      "2023-01-01T00:00:01Z",
			RequestID:      "req-2",
			RPID:           "rp-001",
			ClaimType:      "student_verification",
			PrivacyHash:    "hash-2",
			Status:         "SUCCESS",
			PolicyDecision: "ALLOW",
		},
	}

	// Generate proof
	proof, err := service.GenerateMerkleProof(entries, "req-1")
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	// Test valid proof verification
	leafData := fmt.Sprintf("%s:%s", "req-1", "2023-01-01T00:00:00Z")
	valid := service.VerifyMerkleProof(proof, leafData)
	if !valid {
		t.Error("Expected valid proof to be verified")
	}

	// Test invalid proof verification
	invalidProof := &MerkleProof{
		RootHash:   "invalid_root",
		LeafHash:   proof.LeafHash,
		ProofPath:  proof.ProofPath,
		ProofIndex: proof.ProofIndex,
	}

	valid = service.VerifyMerkleProof(invalidProof, leafData)
	if valid {
		t.Error("Expected invalid proof to fail verification")
	}

	// Test nil proof
	valid = service.VerifyMerkleProof(nil, leafData)
	if valid {
		t.Error("Expected nil proof to fail verification")
	}
}

func TestCreateHashChain(t *testing.T) {
	service := NewCryptographicIntegrityService(&config.Config{})

	previousHash := "previous_hash_123"
	entryID := "entry_456"
	sequence := int64(789)

	chain := service.CreateHashChain(previousHash, entryID, sequence)

	if chain == nil {
		t.Fatal("Expected hash chain to be created")
	}

	if chain.PreviousHash != previousHash {
		t.Error("Expected previous hash to match")
	}

	if chain.EntryID != entryID {
		t.Error("Expected entry ID to match")
	}

	if chain.Sequence != sequence {
		t.Error("Expected sequence to match")
	}

	if chain.CurrentHash == "" {
		t.Error("Expected current hash to be generated")
	}

	if chain.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

func TestVerifyHashChain(t *testing.T) {
	service := NewCryptographicIntegrityService(&config.Config{})

	previousHash := "previous_hash_123"
	entryID := "entry_456"
	sequence := int64(789)

	chain := service.CreateHashChain(previousHash, entryID, sequence)

	// Test valid chain verification
	valid := service.VerifyHashChain(chain)
	if !valid {
		t.Error("Expected valid hash chain to be verified")
	}

	// Test invalid chain verification
	invalidChain := &HashChain{
		PreviousHash: previousHash,
		CurrentHash:  "invalid_hash",
		EntryID:      entryID,
		Sequence:     sequence,
		Timestamp:    chain.Timestamp,
	}

	valid = service.VerifyHashChain(invalidChain)
	if valid {
		t.Error("Expected invalid hash chain to fail verification")
	}

	// Test nil chain
	valid = service.VerifyHashChain(nil)
	if valid {
		t.Error("Expected nil chain to fail verification")
	}
}

func TestGenerateIntegrityHash(t *testing.T) {
	service := NewCryptographicIntegrityService(&config.Config{})

	// Test empty entries
	hash := service.GenerateIntegrityHash([]*models.AuditEntry{})
	if hash != "" {
		t.Error("Expected empty hash for empty entries")
	}

	// Test single entry
	entries := []*models.AuditEntry{
		{
			Timestamp:      "2023-01-01T00:00:00Z",
			RequestID:      "req-1",
			RPID:           "rp-001",
			ClaimType:      "student_verification",
			PrivacyHash:    "hash-1",
			Status:         "SUCCESS",
			PolicyDecision: "ALLOW",
		},
	}

	hash = service.GenerateIntegrityHash(entries)
	if hash == "" {
		t.Fatal("Expected integrity hash to be generated")
	}

	// Test multiple entries
	entries = append(entries, &models.AuditEntry{
		Timestamp:      "2023-01-01T00:00:01Z",
		RequestID:      "req-2",
		RPID:           "rp-001",
		ClaimType:      "student_verification",
		PrivacyHash:    "hash-2",
		Status:         "SUCCESS",
		PolicyDecision: "ALLOW",
	})

	hash2 := service.GenerateIntegrityHash(entries)
	if hash2 == "" {
		t.Fatal("Expected integrity hash to be generated for multiple entries")
	}

	if hash == hash2 {
		t.Error("Expected different hashes for different entry sets")
	}
}

func TestValidateAuditIntegrity(t *testing.T) {
	service := NewCryptographicIntegrityService(&config.Config{})

	// Test empty entries
	err := service.ValidateAuditIntegrity([]*models.AuditEntry{})
	if err == nil {
		t.Error("Expected error for empty entries")
	}

	// Test valid entries
	entries := []*models.AuditEntry{
		{
			Timestamp:      "2023-01-01T00:00:00Z",
			RequestID:      "req-1",
			RPID:           "rp-001",
			ClaimType:      "student_verification",
			PrivacyHash:    "hash-1",
			MerkleProof:    "proof-1",
			Status:         "SUCCESS",
			PolicyDecision: "ALLOW",
		},
		{
			Timestamp:      "2023-01-01T00:00:01Z",
			RequestID:      "req-2",
			RPID:           "rp-001",
			ClaimType:      "student_verification",
			PrivacyHash:    "hash-2",
			MerkleProof:    "proof-2",
			Status:         "SUCCESS",
			PolicyDecision: "ALLOW",
		},
	}

	err = service.ValidateAuditIntegrity(entries)
	if err != nil {
		t.Errorf("Expected no error for valid entries: %v", err)
	}

	// Test duplicate request IDs
	entries[1].RequestID = "req-1"
	err = service.ValidateAuditIntegrity(entries)
	if err == nil {
		t.Error("Expected error for duplicate request IDs")
	}

	// Test invalid entry
	entries[1].RequestID = "req-2"
	entries[1].RequestID = ""
	err = service.ValidateAuditIntegrity(entries)
	if err == nil {
		t.Error("Expected error for invalid entry")
	}
}

func TestCalculateTreeHeight(t *testing.T) {
	service := NewCryptographicIntegrityService(&config.Config{})

	// Test edge cases
	if height := service.calculateTreeHeight(0); height != 0 {
		t.Errorf("Expected height 0 for 0 leaves, got %d", height)
	}

	if height := service.calculateTreeHeight(1); height != 0 {
		t.Errorf("Expected height 0 for 1 leaf, got %d", height)
	}

	if height := service.calculateTreeHeight(2); height != 1 {
		t.Errorf("Expected height 1 for 2 leaves, got %d", height)
	}

	if height := service.calculateTreeHeight(3); height != 2 {
		t.Errorf("Expected height 2 for 3 leaves, got %d", height)
	}

	if height := service.calculateTreeHeight(4); height != 2 {
		t.Errorf("Expected height 2 for 4 leaves, got %d", height)
	}
}

func TestCryptographicIntegrityService_HealthCheck(t *testing.T) {
	service := NewCryptographicIntegrityService(&config.Config{})

	err := service.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("Expected health check to pass: %v", err)
	}
}
