package mcstatus

import (
	"testing"
)

func TestEncodeVarInt(t *testing.T) {
	tests := []struct {
		input    int
		expected []byte
	}{
		{0, []byte{0x00}},
		{1, []byte{0x01}},
		{127, []byte{0x7F}},
		{128, []byte{0x80, 0x01}},
		{255, []byte{0xFF, 0x01}},
		{25565, []byte{0xDD, 0xC7, 0x01}},
	}

	for _, tt := range tests {
		result := encodeVarInt(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("encodeVarInt(%d) = %v, want %v", tt, result, tt.expected)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("encodeVarInt(%d)[%d] = %x, want %x", tt.input, i, result[i], tt.expected[i])
			}
		}
	}
}

func TestEncodeString(t *testing.T) {
	tests := []struct {
		input    string
		expected []byte
	}{
		{"hello", append(encodeVarInt(5), 'h', 'e', 'l', 'l', 'o')},
		{"", encodeVarInt(0)},
	}

	for _, tt := range tests {
		result := encodeString(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("encodeString(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("encodeString(%q)[%d] = %x, want %x", tt.input, i, result[i], tt.expected[i])
			}
		}
	}
}

func TestResolveSRV(t *testing.T) {
	_, _, err := resolveSRV("mc.example.com")
	if err == nil {
		t.Log("SRV resolution succeeded (may vary by network)")
	} else {
		t.Log("SRV resolution failed (expected for non-existent domain)")
	}
}
