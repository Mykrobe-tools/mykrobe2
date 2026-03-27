package probes

import (
	"fmt"
	"io"
)

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func bytesToUpper(seq []byte) []byte {
	out := make([]byte, len(seq))
	for i, b := range seq {
		if b >= 'a' && b <= 'z' {
			out[i] = b - 32
		} else {
			out[i] = b
		}
	}
	return out
}

func closeIfPossible(v any) {
	if c, ok := v.(io.Closer); ok {
		_ = c.Close()
	}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func strconvAtoi(s string) (int, error) {
	var n int
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid integer %q", s)
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}
