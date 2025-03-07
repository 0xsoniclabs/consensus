package cachescale

import (
	"testing"

	"github.com/0xsoniclabs/consensus/inter/idx"
)

func TestRatio_U64(t *testing.T) {
	tests := []struct {
		name string
		r    Ratio
		v    uint64
		want uint64
	}{
		{"identity", Ratio{1, 1}, 5, 5},
		{"scale up exact", Ratio{1, 2}, 5, 10},
		{"scale up with remainder", Ratio{2, 3}, 3, 5},   // (3*3)/2 = 4.5 → 5
		{"scale down exact", Ratio{2, 1}, 4, 2},          // (4*1)/2 = 2
		{"scale down with remainder", Ratio{2, 1}, 5, 3}, // (5*1)/2 = 2.5 → 3
		{"zero value", Ratio{3, 5}, 0, 0},
		{"large numbers", Ratio{100, 50}, 200, 100}, // (200*50)/100 = 100
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.U64(tt.v); got != tt.want {
				t.Errorf("U64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRatio_F32(t *testing.T) {
	tests := []struct {
		name string
		r    Ratio
		v    float32
		want float32
	}{
		{"identity", Identity, 1.5, 1.5},
		{"scale up", Ratio{1, 2}, 1.5, 3.0},
		{"scale down", Ratio{2, 1}, 3.0, 1.5},
		{"non-integer ratio", Ratio{3, 2}, 4.0, 4.0 * 2 / 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.F32(tt.v); got != tt.want {
				t.Errorf("F32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRatio_F64(t *testing.T) {
	tests := []struct {
		name string
		r    Ratio
		v    float64
		want float64
	}{
		{"identity", Identity, 1.5, 1.5},
		{"scale up", Ratio{1, 2}, 1.5, 3.0},
		{"scale down", Ratio{2, 1}, 3.0, 1.5},
		{"non-integer ratio", Ratio{3, 2}, 4.0, 4.0 * 2 / 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.F64(tt.v); got != tt.want {
				t.Errorf("F64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRatio_U(t *testing.T) {
	r := Ratio{Base: 2, Target: 3}
	v := uint(3)
	want := uint(5) // (3*3)/2 = 4.5 → 5
	if got := r.U(v); got != want {
		t.Errorf("U() = %v, want %v", got, want)
	}
}

func TestRatio_U32(t *testing.T) {
	r := Ratio{Base: 3, Target: 2}
	v := uint32(4)
	want := uint32(3) // (4*2)/3 = 2.666... → 3
	if got := r.U32(v); got != want {
		t.Errorf("U32() = %v, want %v", got, want)
	}
}

func TestRatio_I(t *testing.T) {
	r := Ratio{Base: 2, Target: 3}
	v := 3
	want := 5 // (3*3)/2 = 4.5 → 5
	if got := r.I(v); got != want {
		t.Errorf("I() = %v, want %v", got, want)
	}
}

func TestRatio_I32(t *testing.T) {
	r := Ratio{Base: 5, Target: 2}
	v := int32(5)
	want := int32(2) // (5*2)/5 = 2
	if got := r.I32(v); got != want {
		t.Errorf("I32() = %v, want %v", got, want)
	}
}

func TestRatio_I64(t *testing.T) {
	r := Ratio{Base: 3, Target: 4}
	v := int64(3)
	want := int64(4) // (3*4)/3 = 4
	if got := r.I64(v); got != want {
		t.Errorf("I64() = %v, want %v", got, want)
	}
}

func TestRatio_Events(t *testing.T) {
	r := Ratio{Base: 2, Target: 3}
	v := idx.Event(3)
	want := idx.Event(5) // (3*3)/2 = 4.5 → 5
	if got := r.Events(v); got != want {
		t.Errorf("Events() = %v, want %v", got, want)
	}
}

func TestRatio_Blocks(t *testing.T) {
	r := Ratio{Base: 3, Target: 1}
	v := idx.Block(9)
	want := idx.Block(3) // (9*1)/3 = 3
	if got := r.Blocks(v); got != want {
		t.Errorf("Blocks() = %v, want %v", got, want)
	}
}

func TestRatio_Frames(t *testing.T) {
	r := Ratio{Base: 4, Target: 5}
	v := idx.Frame(8)
	want := idx.Frame(10) // (8*5)/4 = 10
	if got := r.Frames(v); got != want {
		t.Errorf("Frames() = %v, want %v", got, want)
	}
}
