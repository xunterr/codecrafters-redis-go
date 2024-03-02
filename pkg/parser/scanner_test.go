package parser

import (
	"testing"
)

func TestScanToken(t *testing.T) {
	tests := []struct {
		input string
		want  TokenType
	}{
		{"$", Dollar},
		{"123", Number},
		{"ABCD", String},
		{"*", Asterisk},
		{"+", Plus},
	}

	for _, e := range tests {
		scanner := NewScanner(e.input)
		scanner.scanToken()
		tokenType := scanner.tokens[0].TokenType
		if tokenType != e.want {
			t.Errorf("Parsed token type doesn`t match input token. Have: %d, want: %d", tokenType, e.want)
		}
	}
}
