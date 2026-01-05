package idempotency

import (
	"strings"
	"testing"
)

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{
			name:    "valid UUID",
			key:     "550e8400-e29b-41d4-a716-446655440000",
			wantErr: nil,
		},
		{
			name:    "valid alphanumeric",
			key:     "abc123-def456_ghi789",
			wantErr: nil,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: ErrKeyRequired,
		},
		{
			name:    "too long",
			key:     strings.Repeat("a", 256),
			wantErr: ErrKeyTooLong,
		},
		{
			name:    "invalid characters - spaces",
			key:     "abc 123",
			wantErr: ErrKeyInvalid,
		},
		{
			name:    "invalid characters - special chars",
			key:     "abc@123",
			wantErr: ErrKeyInvalid,
		},
		{
			name:    "exactly 255 chars",
			key:     strings.Repeat("a", 255),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if err != tt.wantErr {
				t.Errorf("ValidateKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestComputeFingerprint(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{
			name: "empty body",
			body: []byte{},
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name: "simple JSON",
			body: []byte(`{"name":"test"}`),
			want: "bae5ed658ab3546aee12f23f36392f35dba12291", // First 40 chars of SHA256
		},
		{
			name: "same content produces same fingerprint",
			body: []byte(`{"a":1,"b":2}`),
			want: "", // We'll compare in the test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeFingerprint(tt.body)

			// Verify it's a valid hex string
			if len(got) != 64 {
				t.Errorf("ComputeFingerprint() length = %d, want 64", len(got))
			}

			// Verify same input produces same output (deterministic)
			got2 := ComputeFingerprint(tt.body)
			if got != got2 {
				t.Errorf("ComputeFingerprint() not deterministic: %s != %s", got, got2)
			}

			// Verify different inputs produce different outputs
			different := append(tt.body, []byte("extra")...)
			gotDifferent := ComputeFingerprint(different)
			if len(tt.body) > 0 && got == gotDifferent {
				t.Errorf("ComputeFingerprint() same for different inputs")
			}
		})
	}
}

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "already normalized",
			key:  "abc123",
			want: "abc123",
		},
		{
			name: "leading spaces",
			key:  "  abc123",
			want: "abc123",
		},
		{
			name: "trailing spaces",
			key:  "abc123  ",
			want: "abc123",
		},
		{
			name: "both sides",
			key:  "  abc123  ",
			want: "abc123",
		},
		{
			name: "tabs",
			key:  "\tabc123\t",
			want: "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeKey(tt.key)
			if got != tt.want {
				t.Errorf("NormalizeKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidKeyChar(t *testing.T) {
	tests := []struct {
		name string
		char rune
		want bool
	}{
		{"lowercase letter", 'a', true},
		{"uppercase letter", 'A', true},
		{"digit", '5', true},
		{"hyphen", '-', true},
		{"underscore", '_', true},
		{"space", ' ', false},
		{"at sign", '@', false},
		{"period", '.', false},
		{"slash", '/', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidKeyChar(tt.char)
			if got != tt.want {
				t.Errorf("IsValidKeyChar(%c) = %v, want %v", tt.char, got, tt.want)
			}
		})
	}
}

func BenchmarkComputeFingerprint(b *testing.B) {
	body := []byte(`{"orderId":"ORD-123","items":[{"sku":"ITEM-1","qty":5}],"customer":"CUST-456"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ComputeFingerprint(body)
	}
}

func BenchmarkValidateKey(b *testing.B) {
	key := "550e8400-e29b-41d4-a716-446655440000"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateKey(key)
	}
}
