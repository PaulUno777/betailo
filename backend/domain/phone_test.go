package domain

import (
	"errors"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	valid := map[string]string{
		"6 90 12 34 56":     "+237690123456",
		"690-12-34-56":      "+237690123456",
		"690123456":         "+237690123456",
		"+237 690 12 34 56": "+237690123456",
		"00237690123456":    "+237690123456",
		"(+237) 222123456":  "+237222123456",
		"+33 6 12 34 56 78": "+33612345678",
	}
	for in, want := range valid {
		got, err := NormalizePhone(in)
		if err != nil || got != want {
			t.Errorf("NormalizePhone(%q) = %q, %v; want %q", in, got, err, want)
		}
	}

	for _, in := range []string{"", "12345", "590123456", "+2376901234", "6901234567", "+33 12", "69O123456", "690+123456"} {
		if _, err := NormalizePhone(in); !errors.Is(err, ErrInvalid) {
			t.Errorf("NormalizePhone(%q) error = %v, want ErrInvalid", in, err)
		}
	}
}
