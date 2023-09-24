package chunkedbody

import (
	"strings"
	"testing"
)

func BenchmarkChunkedBodyParser(b *testing.B) {
	chunkedEnd := "0\r\n\r\n"
	smallChunked := []byte(
		"d\r\nHello, world!\r\n1a\r\nBut what's wrong with you?\r\nf\r\nFinally am here\r\n" + chunkedEnd,
	)
	mediumChunked := []byte(
		strings.Repeat("1a\r\nBut what's wrong with you?\r\n", 15) + chunkedEnd,
	)
	bigChunked := []byte(
		strings.Repeat("1a\r\nBut what's wrong with you?\r\n", 100) + chunkedEnd,
	)

	parser := NewChunkedBodyParser(DefaultSettings())

	b.Run("small with 3 chunks", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = parser.Parse(smallChunked, false)
		}
	})

	b.Run("medium with 15 chunks", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = parser.Parse(mediumChunked, false)
		}
	})

	b.Run("big with 100 chunks", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = parser.Parse(bigChunked, false)
		}
	})
}
