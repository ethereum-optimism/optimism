package proxyd

import (
	"testing"
)

func Test_blockToFloat(t *testing.T) {
	type args struct {
		hexVal string
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{"0xf1b3", args{"0xf1b3"}, float64(61875)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := blockToFloat(tt.args.hexVal); got != tt.want {
				t.Errorf("blockToFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hexAdd(t *testing.T) {
	type args struct {
		hexVal string
		incr   int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"0x1", args{"0x1", 1}, "0x2"},
		{"0x2", args{"0x2", -1}, "0x1"},
		{"0xf", args{"0xf", 1}, "0x10"},
		{"0x10", args{"0x10", -1}, "0xf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hexAdd(tt.args.hexVal, tt.args.incr); got != tt.want {
				t.Errorf("hexAdd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_blockAheadOrEqual(t *testing.T) {
	type args struct {
		baseBlock  string
		checkBlock string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"0x1 vs 0x1", args{"0x1", "0x1"}, true},
		{"0x2 vs 0x1", args{"0x2", "0x1"}, true},
		{"0x1 vs 0x2", args{"0x1", "0x2"}, false},
		{"0xff vs 0x100", args{"0xff", "0x100"}, false},
		{"0x100 vs 0xff", args{"0x100", "0xff"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := blockAheadOrEqual(tt.args.baseBlock, tt.args.checkBlock); got != tt.want {
				t.Errorf("blockAheadOrEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
