package main

import (
	"math/big"
	"testing"
)

func TestParseEtherAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "one ether", input: "1", want: "1000000000000000000"},
		{name: "fractional ether", input: "0.001", want: "1000000000000000"},
		{name: "eighteen decimal places", input: "0.000000000000000001", want: "1"},
		{name: "too many decimal places", input: "0.0000000000000000001", wantErr: true},
		{name: "zero", input: "0", wantErr: true},
		{name: "not a number", input: "abc", wantErr: true},
		{name: "fraction notation", input: "1/2", wantErr: true},
		{name: "exponent notation", input: "1e-3", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseEtherAmount(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseEtherAmount(%q) error = %v, wantErr %v", test.input, err, test.wantErr)
			}
			if err == nil {
				want, ok := new(big.Int).SetString(test.want, 10)
				if !ok {
					t.Fatalf("invalid test expectation %q", test.want)
				}
				if got.Cmp(want) != 0 {
					t.Errorf("parseEtherAmount(%q) = %s, want %s", test.input, got, test.want)
				}
			}
		})
	}
}