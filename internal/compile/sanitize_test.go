package compile

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

func TestSanitizeXMLBytes(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{
			name: "pure ASCII passes through",
			in:   []byte(`<smi><module name="FOO"/></smi>`),
			want: []byte(`<smi><module name="FOO"/></smi>`),
		},
		{
			name: "empty input",
			in:   []byte{},
			want: []byte{},
		},
		{
			name: "Latin-1 degree sign expands to UTF-8",
			in:   []byte{0xB0},
			want: []byte{0xC2, 0xB0},
		},
		{
			name: "Latin-1 micro sign expands to UTF-8",
			in:   []byte{0xB5},
			want: []byte{0xC2, 0xB5},
		},
		{
			name: "backspace 0x08 is dropped",
			in:   []byte{'A', 0x08, 'B'},
			want: []byte{'A', 'B'},
		},
		{
			name: "Tab LF CR are preserved verbatim",
			in:   []byte{'X', 0x09, 'Y', 0x0A, 'Z', 0x0D, 'W'},
			want: []byte{'X', 0x09, 'Y', 0x0A, 'Z', 0x0D, 'W'},
		},
		{
			name: "all forbidden C0 controls are dropped, allowed ones kept",
			in: []byte{
				0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, // kept
				0x0A, // kept
				0x0B, 0x0C,
				0x0D, // kept
				0x0E, 0x0F, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
				0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D,
				0x1E, 0x1F,
			},
			want: []byte{0x09, 0x0A, 0x0D},
		},
		{
			name: "DEL 0x7F passes through (XML-permitted)",
			in:   []byte{'A', 0x7F, 'B'},
			want: []byte{'A', 0x7F, 'B'},
		},
		{
			name: "0x80 boundary expands correctly",
			in:   []byte{0x80},
			want: []byte{0xC2, 0x80},
		},
		{
			name: "0xFF boundary expands correctly",
			in:   []byte{0xFF},
			want: []byte{0xC3, 0xBF},
		},
		{
			name: "mixed Latin-1 + control + ASCII in one pass",
			in: []byte{
				'<', 'd', '>',
				'2', '5', 0xB0, 'C',
				0x08,
				0x09, 'x',
				'<', '/', 'd', '>',
			},
			want: []byte{
				'<', 'd', '>',
				'2', '5', 0xC2, 0xB0, 'C',
				0x09, 'x',
				'<', '/', 'd', '>',
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeXMLBytes(tt.in)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("sanitizeXMLBytes(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeXMLBytesDoesNotMutateInput(t *testing.T) {
	in := []byte{'A', 0xB0, 0x08, 'B'}
	original := append([]byte{}, in...)
	_ = sanitizeXMLBytes(in)
	if !bytes.Equal(in, original) {
		t.Errorf("sanitizeXMLBytes mutated its input: got %v, want %v", in, original)
	}
}

func TestXMLNeedsSanitize(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "invalid UTF-8 unwrapped",
			err:  errors.New("XML syntax error on line 42: invalid UTF-8"),
			want: true,
		},
		{
			name: "illegal character code unwrapped",
			err:  errors.New("XML syntax error on line 7: illegal character code U+0008"),
			want: true,
		},
		{
			name: "invalid UTF-8 wrapped via fmt.Errorf %w",
			err:  fmt.Errorf("parse smidump output for foo: %w", errors.New("XML syntax error on line 42: invalid UTF-8")),
			want: true,
		},
		{
			name: "illegal character code wrapped",
			err:  fmt.Errorf("parse smidump output for foo: %w", errors.New("XML syntax error on line 7: illegal character code U+0008")),
			want: true,
		},
		{
			name: "unexpected EOF does not match",
			err:  errors.New("XML syntax error on line 99: unexpected EOF"),
			want: false,
		},
		{
			name: "element not closed does not match",
			err:  errors.New("XML syntax error: element <foo> not closed"),
			want: false,
		},
		{
			name: "unknown entity does not match",
			err:  errors.New("XML syntax error: unknown entity &bogus;"),
			want: false,
		},
		{
			name: "wrapped EOF still does not match",
			err:  fmt.Errorf("parse smidump output for bar: %w", errors.New("unexpected EOF")),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := xmlNeedsSanitize(tt.err); got != tt.want {
				t.Errorf("xmlNeedsSanitize(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
