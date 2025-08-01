package services

import (
	"math"
	"testing"
)

func TestNewBloomFilter(t *testing.T) {
	size := 1000
	hashCount := 5
	bf := NewBloomFilter(size, hashCount)
	
	if bf.size != size {
		t.Errorf("Expected size %d, got %d", size, bf.size)
	}
	
	if bf.hashCount != hashCount {
		t.Errorf("Expected hash count %d, got %d", hashCount, bf.hashCount)
	}
	
	if len(bf.bitArray) != size {
		t.Errorf("Expected bit array length %d, got %d", size, len(bf.bitArray))
	}
	
	if len(bf.hashFunctions) != hashCount {
		t.Errorf("Expected hash functions count %d, got %d", hashCount, len(bf.hashFunctions))
	}
}

func TestBloomFilter_AddAndContains(t *testing.T) {
	bf := NewBloomFilter(1000, 5)
	
	// Test adding and checking elements
	testElements := []string{"test1", "test2", "test3", "hello world", "john doe"}
	
	for _, element := range testElements {
		bf.Add(element)
		
		if !bf.Contains(element) {
			t.Errorf("Bloom filter should contain element: %s", element)
		}
	}
	
	// Test that non-added elements are not contained
	nonElements := []string{"not_added", "different", "another_test"}
	for _, element := range nonElements {
		if bf.Contains(element) {
			t.Errorf("Bloom filter should not contain element: %s", element)
		}
	}
}

func TestBloomFilter_ToHexString(t *testing.T) {
	bf := NewBloomFilter(100, 3)
	bf.Add("test")
	
	hexString := bf.ToHexString()
	
	if hexString == "" {
		t.Error("Hex string should not be empty")
	}
	
	// Test that hex string is valid
	if len(hexString)%2 != 0 {
		t.Error("Hex string should have even length")
	}
}

func TestBloomFilter_FromHexString(t *testing.T) {
	original := NewBloomFilter(100, 3)
	original.Add("test")
	original.Add("hello")
	
	hexString := original.ToHexString()
	
	// Recreate from hex string
	recreated, err := FromHexString(hexString, 100, 3)
	if err != nil {
		t.Fatalf("Failed to create Bloom filter from hex string: %v", err)
	}
	
	// Test that recreated filter contains the same elements
	if !recreated.Contains("test") {
		t.Error("Recreated Bloom filter should contain 'test'")
	}
	
	if !recreated.Contains("hello") {
		t.Error("Recreated Bloom filter should contain 'hello'")
	}
}

func TestBloomFilter_GetFalsePositiveRate(t *testing.T) {
	bf := NewBloomFilter(1000, 5)
	
	// Empty filter should have 0 false positive rate
	rate := bf.GetFalsePositiveRate()
	if rate != 0 {
		t.Errorf("Empty Bloom filter should have 0 false positive rate, got %f", rate)
	}
	
	// Add some elements
	bf.Add("test1")
	bf.Add("test2")
	
	// Should have some false positive rate
	rate = bf.GetFalsePositiveRate()
	if rate < 0 || rate > 1 {
		t.Errorf("False positive rate should be between 0 and 1, got %f", rate)
	}
}

func TestPhoneticEncoder_Encode(t *testing.T) {
	encoder := NewPhoneticEncoder()
	
	testCases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"John", "J500"},
		{"Smith", "S530"},
		{"Johnson", "J525"},
		{"Williams", "W452"},
		{"Brown", "B650"},
		{"Davis", "D120"},
		{"Miller", "M460"},
		{"Wilson", "W425"},
		{"Moore", "M600"},
		{"Taylor", "T460"},
		{"Anderson", "A536"},
		{"Thomas", "T520"},
		{"Jackson", "J250"},
		{"White", "W300"},
		{"Jane", "J500"}, // Same as John due to no consonants
	}
	
	for _, tc := range testCases {
		result := encoder.Encode(tc.input)
		if result != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, result)
		}
	}
}

func TestFuzzyMatcher_CalculateSimilarity(t *testing.T) {
	matcher := NewFuzzyMatcher()
	
	testCases := []struct {
		str1     string
		str2     string
		expected float64
	}{
		{"", "", 0.0}, // Empty strings should return 0.0
		{"hello", "", 0.0},
		{"", "world", 0.0},
		{"hello", "hello", 1.0},
		{"hello", "helo", 0.8}, // 1 edit distance
		{"hello", "world", 0.2}, // 4 edit distance, normalized
		{"john", "jon", 0.75},   // 1 edit distance
		{"smith", "smyth", 0.8}, // 1 edit distance
	}
	
	for _, tc := range testCases {
		result := matcher.CalculateSimilarity(tc.str1, tc.str2)
		// Use approximate comparison for floating point values
		if math.Abs(result-tc.expected) > 0.001 {
			t.Errorf("For '%s' vs '%s', expected %f, got %f", tc.str1, tc.str2, tc.expected, result)
		}
	}
}

func TestFuzzyMatcher_IsPhoneticallySimilar(t *testing.T) {
	matcher := NewFuzzyMatcher()
	
	testCases := []struct {
		name1    string
		name2    string
		expected bool
	}{
		{"John", "Jon", true},      // Both encode to J500
		{"Smith", "Smyth", true},   // Both encode to S530
		{"Johnson", "Jonson", true}, // Both encode to J525
		{"John", "Jane", true},     // Both encode to J500 (no consonants)
		{"Smith", "Brown", false},  // Different encodings: S530 vs B650
	}
	
	for _, tc := range testCases {
		result := matcher.IsPhoneticallySimilar(tc.name1, tc.name2)
		if result != tc.expected {
			t.Errorf("For '%s' vs '%s', expected %t, got %t", tc.name1, tc.name2, tc.expected, result)
		}
	}
}

func TestFuzzyMatcher_GetPhoneticCode(t *testing.T) {
	matcher := NewFuzzyMatcher()
	
	testCases := []struct {
		input    string
		expected string
	}{
		{"John", "J500"},
		{"Smith", "S530"},
		{"Johnson", "J525"},
		{"Williams", "W452"},
		{"Brown", "B650"},
		{"Jane", "J500"}, // Same as John due to no consonants
	}
	
	for _, tc := range testCases {
		result := matcher.GetPhoneticCode(tc.input)
		if result != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, result)
		}
	}
}

func TestFuzzyMatcher_LevenshteinDistance(t *testing.T) {
	matcher := NewFuzzyMatcher()
	
	testCases := []struct {
		str1     string
		str2     string
		expected int
	}{
		{"", "", 0},
		{"hello", "", 5},
		{"", "world", 5},
		{"hello", "hello", 0},
		{"hello", "helo", 1},
		{"hello", "world", 4},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
	}
	
	for _, tc := range testCases {
		result := matcher.levenshteinDistance(tc.str1, tc.str2)
		if result != tc.expected {
			t.Errorf("For '%s' vs '%s', expected distance %d, got %d", tc.str1, tc.str2, tc.expected, result)
		}
	}
} 