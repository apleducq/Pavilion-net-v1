package services

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
)

// BloomFilter represents a Bloom filter for privacy-preserving record linkage
type BloomFilter struct {
	bitArray    []bool
	size        int
	hashCount   int
	hashFunctions []func([]byte) uint64
}

// NewBloomFilter creates a new Bloom filter with the specified parameters
func NewBloomFilter(size, hashCount int) *BloomFilter {
	bf := &BloomFilter{
		bitArray:  make([]bool, size),
		size:      size,
		hashCount: hashCount,
	}
	
	// Initialize hash functions
	bf.hashFunctions = bf.createHashFunctions()
	
	return bf
}

// createHashFunctions creates multiple hash functions for the Bloom filter
func (bf *BloomFilter) createHashFunctions() []func([]byte) uint64 {
	functions := make([]func([]byte) uint64, bf.hashCount)
	
	for i := 0; i < bf.hashCount; i++ {
		seed := uint64(i)
		functions[i] = func(data []byte) uint64 {
			// Use FNV-1a hash with different seeds
			h := fnv.New64a()
			h.Write([]byte(fmt.Sprintf("%d", seed)))
			h.Write(data)
			return h.Sum64()
		}
	}
	
	return functions
}

// Add adds an element to the Bloom filter
func (bf *BloomFilter) Add(element string) {
	data := []byte(strings.ToLower(element))
	
	for _, hashFunc := range bf.hashFunctions {
		hash := hashFunc(data)
		index := hash % uint64(bf.size)
		bf.bitArray[index] = true
	}
}

// Contains checks if an element might be in the Bloom filter
func (bf *BloomFilter) Contains(element string) bool {
	data := []byte(strings.ToLower(element))
	
	for _, hashFunc := range bf.hashFunctions {
		hash := hashFunc(data)
		index := hash % uint64(bf.size)
		if !bf.bitArray[index] {
			return false
		}
	}
	return true
}

// ToHexString converts the Bloom filter to a hex string for transmission
func (bf *BloomFilter) ToHexString() string {
	// Convert boolean array to bytes
	byteArray := make([]byte, (bf.size+7)/8) // Ceiling division
	
	for i := 0; i < bf.size; i++ {
		if bf.bitArray[i] {
			byteIndex := i / 8
			bitIndex := i % 8
			byteArray[byteIndex] |= 1 << bitIndex
		}
	}
	
	return hex.EncodeToString(byteArray)
}

// FromHexString creates a Bloom filter from a hex string
func FromHexString(hexString string, size, hashCount int) (*BloomFilter, error) {
	byteArray, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string: %w", err)
	}
	
	bf := NewBloomFilter(size, hashCount)
	
	// Convert bytes back to boolean array
	for i := 0; i < bf.size; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if byteIndex < len(byteArray) {
			bf.bitArray[i] = (byteArray[byteIndex] & (1 << bitIndex)) != 0
		}
	}
	
	return bf, nil
}

// GetFalsePositiveRate calculates the theoretical false positive rate
func (bf *BloomFilter) GetFalsePositiveRate() float64 {
	// Count set bits
	setBits := 0
	for _, bit := range bf.bitArray {
		if bit {
			setBits++
		}
	}
	
	// Calculate probability of a bit being set
	p := float64(setBits) / float64(bf.size)
	
	// False positive rate = p^hashCount
	return math.Pow(p, float64(bf.hashCount))
}

// PhoneticEncoder provides phonetic encoding for fuzzy name matching
type PhoneticEncoder struct{}

// NewPhoneticEncoder creates a new phonetic encoder
func NewPhoneticEncoder() *PhoneticEncoder {
	return &PhoneticEncoder{}
}

// Encode applies phonetic encoding to a name (Soundex-like algorithm)
func (pe *PhoneticEncoder) Encode(name string) string {
	if name == "" {
		return ""
	}
	
	// Convert to uppercase and remove non-alphabetic characters
	name = strings.ToUpper(name)
	name = strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r
		}
		return -1
	}, name)
	
	if name == "" {
		return ""
	}
	
	// Standard Soundex-like encoding
	encoded := string(name[0]) // Keep first letter
	
	// Map similar sounds to the same digit
	soundMap := map[rune]string{
		'B': "1", 'F': "1", 'P': "1", 'V': "1",
		'C': "2", 'G': "2", 'J': "2", 'K': "2", 'Q': "2", 'S': "2", 'X': "2", 'Z': "2",
		'D': "3", 'T': "3",
		'L': "4",
		'M': "5", 'N': "5",
		'R': "6",
	}
	
	prevDigit := ""
	for _, char := range name[1:] {
		if digit, exists := soundMap[char]; exists && digit != prevDigit {
			encoded += digit
			prevDigit = digit
		}
		// Stop after 3 digits (4 characters total including first letter)
		if len(encoded) >= 4 {
			break
		}
	}
	
	// Pad to 4 characters
	for len(encoded) < 4 {
		encoded += "0"
	}
	
	return encoded[:4]
}

// FuzzyMatcher provides fuzzy matching capabilities for names
type FuzzyMatcher struct {
	encoder *PhoneticEncoder
}

// NewFuzzyMatcher creates a new fuzzy matcher
func NewFuzzyMatcher() *FuzzyMatcher {
	return &FuzzyMatcher{
		encoder: NewPhoneticEncoder(),
	}
}

// CalculateSimilarity calculates similarity between two strings
func (fm *FuzzyMatcher) CalculateSimilarity(str1, str2 string) float64 {
	if str1 == str2 {
		// If both are empty, return 0.0, otherwise return 1.0
		if str1 == "" {
			return 0.0
		}
		return 1.0
	}
	
	if str1 == "" || str2 == "" {
		return 0.0
	}
	
	// Convert to lowercase for comparison
	str1 = strings.ToLower(str1)
	str2 = strings.ToLower(str2)
	
	// Calculate Levenshtein distance
	distance := fm.levenshteinDistance(str1, str2)
	maxLen := float64(max(len(str1), len(str2)))
	
	// Return similarity as 1 - normalized distance
	return 1.0 - (float64(distance) / maxLen)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func (fm *FuzzyMatcher) levenshteinDistance(str1, str2 string) int {
	len1, len2 := len(str1), len(str2)
	
	// Create matrix
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}
	
	// Initialize first row and column
	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}
	
	// Fill matrix
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if str1[i-1] != str2[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,    // deletion
				min(
					matrix[i][j-1]+1,    // insertion
					matrix[i-1][j-1]+cost, // substitution
				),
			)
		}
	}
	
	return matrix[len1][len2]
}

// GetPhoneticCode returns the phonetic encoding of a name
func (fm *FuzzyMatcher) GetPhoneticCode(name string) string {
	return fm.encoder.Encode(name)
}

// IsPhoneticallySimilar checks if two names are phonetically similar
func (fm *FuzzyMatcher) IsPhoneticallySimilar(name1, name2 string) bool {
	code1 := fm.encoder.Encode(name1)
	code2 := fm.encoder.Encode(name2)
	return code1 == code2
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
} 