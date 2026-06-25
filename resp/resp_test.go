package resp

import (
	"errors"
	"strings"
	"testing"
)

func BenchmarkEncode(b *testing.B) {
	cases := []struct {
		name     string
		value    any
		isSimple bool
	}{
		{"nil", nil, false},
		{"string_simple", "hello world", true},
		{"string_bulk", "hello world", false},
		{"string_long", strings.Repeat("x", 100), false},
		{"int", 12345, false},
		{"int64", int64(12345), false},
		{"error", errors.New("ERR something went wrong"), false},
		{"string_slice", []string{"foo", "bar", "baz"}, false},
		{"any_slice", []any{"foo", 42, "bar"}, false},
	}

	buf := make([]byte, 0, 4096)

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Encode(buf[:0], tc.value, tc.isSimple)
			}
		})
	}

	_ = buf
}
