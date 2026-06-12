package tools

import "testing"

func TestEvalSimple(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want float64
	}{
		{name: "add", expr: "1+2", want: 3},
		{name: "subtract", expr: "10 - 3", want: 7},
		{name: "multiply", expr: "23*7", want: 161},
		{name: "divide", expr: "8 / 2", want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evalSimple(tt.expr)
			if err != nil {
				t.Fatalf("evalSimple() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("evalSimple() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalSimpleDivisionByZero(t *testing.T) {
	_, err := evalSimple("1/0")
	if err == nil {
		t.Fatal("evalSimple() expected division by zero error")
	}
}
