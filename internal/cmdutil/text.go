package cmdutil

import (
	"bytes"
	"encoding/binary"
	"unicode/utf16"
)

// NormalizeText converts JSON read from a file or stdin to plain UTF-8.
//
// Windows tools often write text with a byte order mark: Windows PowerShell
// 5.1's Out-File and > redirection produce UTF-16 LE, and Set-Content
// -Encoding utf8 prepends a UTF-8 BOM. encoding/json accepts neither, so a
// leading BOM is stripped and UTF-16 is transcoded. Input without a BOM is
// returned unchanged.
func NormalizeText(b []byte) []byte {
	switch {
	case bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}):
		return b[3:]
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}):
		return decodeUTF16(b[2:], binary.LittleEndian)
	case bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		return decodeUTF16(b[2:], binary.BigEndian)
	}
	return b
}

func decodeUTF16(b []byte, order binary.ByteOrder) []byte {
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = order.Uint16(b[2*i:])
	}
	return []byte(string(utf16.Decode(units)))
}
