package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVersion(t *testing.T) {
	var tests = []struct {
		name_ string
		input string
		want  struct {
			op       int
			clearVer string
		}
	}{
		{
			name_: "1",
			input: ">=1.0",
			want: struct {
				op       int
				clearVer string
			}{
				op:       MORE_EQUAL_THEN,
				clearVer: "1.0",
			},
		},
		{
			name_: "2",
			input: "<10",
			want: struct {
				op       int
				clearVer string
			}{
				op:       LESS_THEN,
				clearVer: "10",
			},
		},
		{
			name_: "3",
			input: "",
			want: struct {
				op       int
				clearVer string
			}{
				op:       ALL,
				clearVer: "",
			},
		},
	}
	for _, tt := range tests {
		op, clear_ := ParseVersion(tt.input)
		if !assert.Equal(t, tt.want.op, op) || !assert.Equal(t, tt.want.clearVer, clear_) {
			t.Fatal("responses is not equal")
		}
	}
}

func TestIsApplyVersion(t *testing.T) {
	var tests = []struct {
		name_ string
		one   string
		two   string
		op    int
		want  bool
	}{
		{
			name_: "0",
			one:   "16.2",
			two:   "0.11",
			op:    MORE_EQUAL_THEN,
			want:  true,
		},
		{
			name_: "1",
			one:   "16.2",
			two:   "0.11",
			op:    LESS_THEN,
			want:  false,
		},
		{
			name_: "2",
			one:   "0.11",
			two:   "0.11",
			op:    MORE_EQUAL_THEN,
			want:  true,
		},
	}
	for _, tt := range tests {
		ok, err := CompareVersions(tt.one, tt.two, tt.op)
		if err != nil {
			t.Fatal(err)
		}
		if !assert.Equal(t, tt.want, ok) {
			t.Fatal("responses is not equal")
		}
	}
}
