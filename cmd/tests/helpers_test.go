package tests

import (
	"testing"
)

func TestSliceEqual(t *testing.T) {
	testCases := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "empty slices",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
		{
			name:     "nil slices",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "one nil, one empty",
			a:        nil,
			b:        []string{},
			expected: true,
		},
		{
			name:     "same content",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "different length",
			a:        []string{"a", "b"},
			b:        []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "different content",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "d"},
			expected: false,
		},
		{
			name:     "different order",
			a:        []string{"a", "b", "c"},
			b:        []string{"b", "a", "c"},
			expected: false,
		},
		{
			name:     "case sensitive",
			a:        []string{"A", "B", "C"},
			b:        []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "with empty strings",
			a:        []string{"", "b", ""},
			b:        []string{"", "b", ""},
			expected: true,
		},
		{
			name:     "with spaces",
			a:        []string{" ", "b", " c "},
			b:        []string{" ", "b", " c "},
			expected: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := sliceEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("SliceEqual(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
