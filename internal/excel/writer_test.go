package excel

import "testing"

func TestSafeSheetName(t *testing.T) {
	used := map[string]bool{}
	got := SafeSheetName("A:B/C?D*E[F]G", 1, used)
	want := "A_B_C_D_E_F_G"
	if got != want {
		t.Fatalf("SafeSheetName() = %q, want %q", got, want)
	}

	dup := SafeSheetName("A:B/C?D*E[F]G", 2, used)
	if dup != "A_B_C_D_E_F_G_2" {
		t.Fatalf("duplicate sheet name = %q", dup)
	}
}
