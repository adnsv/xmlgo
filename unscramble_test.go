package xg

import "testing"

func Test_unscramble(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{"empty", "", ""},
		{"trivial 1", "abc", "abc"},
		{"lt", "&lt;", "<"},
		{"gt", "&gt;", ">"},
		{"amp", "&amp;", "&"},
		{"apos", "&apos;", "'"},
		{"quote", "&quot;", "\""},
		{"unknown", "&unknown;", "&unknown;"},
		{"unterminated lt", "&lt", "&lt"},
		{"hex cp", "&#x20;", " "},
		{"dec cp", "&#32;", " "},
		{"hex cp 16", "&#xffff;", "\uffff"},
		{"hex cp max", "&#x10ffff;", "\U0010ffff"},
		{"hex too large", "&#x110000;", "&#x110000;"},
		{"mixed 1", "abc &lt; def &gt; ghi", "abc < def > ghi"},
		{"mixed invalid", "abc &lt def &gt ghi", "abc &lt def &gt ghi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := unscramble(tt.arg); got != tt.want {
				t.Errorf("unscramble() = %v, want %v", got, tt.want)
			}
		})
	}
}
