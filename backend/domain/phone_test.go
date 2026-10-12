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

func TestMaskPhone(t *testing.T) {
	if got := MaskPhone("+237690123456"); got != "+237 6•• •• 34 56" {
		t.Errorf("MaskPhone(cameroon) = %q", got)
	}
	if got := MaskPhone("+33612345678"); got != "•••• 5678" {
		t.Errorf("MaskPhone(foreign) = %q", got)
	}
}

func TestIsCameroonMobile(t *testing.T) {
	for in, want := range map[string]bool{"+237690123456": true, "+237222123456": false, "+33612345678": false} {
		if got := IsCameroonMobile(in); got != want {
			t.Errorf("IsCameroonMobile(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestFormatReference(t *testing.T) {
	if got := FormatReference(ReservationRefPrefix, 42); got != "RSV-0042" {
		t.Errorf("FormatReference = %q", got)
	}
	if got := FormatReference(CycleRefPrefix, 12345); got != "CYC-12345" {
		t.Errorf("FormatReference = %q", got)
	}
}
