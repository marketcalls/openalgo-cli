package cmdutil

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func utf16Bytes(s string, bom []byte, order binary.AppendByteOrder) []byte {
	out := append([]byte{}, bom...)
	for _, u := range utf16.Encode([]rune(s)) {
		out = order.AppendUint16(out, u)
	}
	return out
}

func TestNormalizeText(t *testing.T) {
	const want = `[{"symbol":"SBIN","qty":1}]`
	cases := map[string][]byte{
		"plain":    []byte(want),
		"utf8 bom": append([]byte{0xEF, 0xBB, 0xBF}, want...),
		"utf16 le": utf16Bytes(want, []byte{0xFF, 0xFE}, binary.LittleEndian),
		"utf16 be": utf16Bytes(want, []byte{0xFE, 0xFF}, binary.BigEndian),
	}
	for name, in := range cases {
		if got := string(NormalizeText(in)); got != want {
			t.Errorf("%s: NormalizeText = %q, want %q", name, got, want)
		}
	}
}
