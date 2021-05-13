package xg

import (
	"strings"
	"unicode/utf8"
)

func CalcLocation(buf string, byteoffset int) (line, pos int) {
	buf = strings.TrimPrefix(buf, "\xef\xbb\xbf") // skip bom

	cur, end := 0, len(buf)
	linestart := cur

	if byteoffset > end {
		byteoffset = end
	}

	for cur < byteoffset {
		c := buf[cur]
		cur++
		if c == '\n' {
			line++
			linestart = cur
		} else if c == '\r' {
			if cur < byteoffset && buf[cur] == '\n' {
				cur++
			}
			line++
			linestart = cur
		}
	}
	pos = utf8.RuneCountInString(buf[linestart:byteoffset])

	return
}
