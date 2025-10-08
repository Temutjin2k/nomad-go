package hasher

import "testing"

func TestSum_Deterministic(t *testing.T) {
	in := "same input"
	h1 := Hash(in)
	h2 := Hash(in)
	if h1 != h2 {
		t.Fatalf("hash must be deterministic, got %s vs %s", h1, h2)
	}
}

func TestSum_DifferentInputs(t *testing.T) {
	if Hash("a") == Hash("b") {
		t.Fatalf("different inputs should not produce the same hash")
	}
}

func TestSum_KnownVector(t *testing.T) {
	// SHA-256("hello") = 2cf24d... per стандартным векторам
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	got := Hash("hello")
	if got != want {
		t.Fatalf("unexpected hash: got %s want %s", got, want)
	}
}

func BenchmarkSum(b *testing.B) {
	in := "some reasonably sized input"

	for b.Loop() {
		_ = Hash(in)
	}
}
