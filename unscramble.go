package xg

import (
	"strconv"
	"strings"
)

func extractcp(s string) (cp rune, n int) {
	n = strings.IndexByte(s, ';')
	if n < 2 {
		return 0, 0
	}

	if s[0] == '#' {
		if s[1] == 'x' {
			v, err := strconv.ParseUint(s[2:n], 16, 32)
			if err != nil || v > '\U0010ffff' {
				return 0, 0
			}
			return rune(v), n + 1
		} else {
			v, err := strconv.ParseUint(s[1:n], 10, 32)
			if err != nil || v > '\U0010ffff' {
				return 0, 0
			}
			return rune(v), n + 1
		}
	}

	switch s[:n] {
	case "lt":
		return '<', n + 1
	case "gt":
		return '>', n + 1
	case "amp":
		return '&', n + 1
	case "apos":
		return '\'', n + 1
	case "quot":
		return '"', n + 1
	default:
		return 0, 0
	}
}

func unscramble(s string) string {
	i := strings.IndexByte(s, '&')
	if i < 0 {
		return s
	}
	r := s[:i]
	s = s[i:]
	for {
		i := strings.IndexByte(s, '&')
		if i < 0 {
			r += s
			break
		}
		r += s[:i]
		s = s[i+1:]

		cp, n := extractcp(s)
		if n == 0 {
			r += "&"
		} else {
			r += string(cp)
			s = s[n:]
		}
	}
	return r
}
