package exitcode

import (
	"errors"
	"testing"
)

func TestExitStatus_Error(t *testing.T) {
	tests := []struct {
		name string
		s    ExitStatus
		want string
	}{
		{"zero", ExitStatusOK, "exit status 0"},
		{"non-zero", ExitStatus(3), "exit status 3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Error(); got != tt.want {
				t.Errorf("ExitStatus(%d).Error() = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestNewExitStatus_RoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		status uint8
	}{
		{"zero", 0},
		{"one", 1},
		{"three", 3},
		{"max", 255},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewExitStatus(tt.status)

			var es ExitStatus
			if !errors.As(err, &es) {
				t.Fatalf("errors.As(%v) failed to extract ExitStatus", err)
			}
			if uint8(es) != tt.status {
				t.Errorf("round-tripped status = %d, want %d", uint8(es), tt.status)
			}
		})
	}
}

func TestExitStatus_errorsAs_Extraction(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		want   uint8
		wantOK bool
	}{
		{"zero", NewExitStatus(0), 0, true},
		{"non-zero", NewExitStatus(7), 7, true},
		{"ordinary error", errors.New("boom"), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var es ExitStatus
			ok := errors.As(tt.err, &es)
			if ok != tt.wantOK {
				t.Fatalf("errors.As ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && uint8(es) != tt.want {
				t.Errorf("errors.As extracted status = %d, want %d", uint8(es), tt.want)
			}
		})
	}
}

func TestExitStatus_errorsAsType_Extraction(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		want   uint8
		wantOK bool
	}{
		{"zero", NewExitStatus(0), 0, true},
		{"non-zero", NewExitStatus(9), 9, true},
		{"ordinary error", errors.New("boom"), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es, ok := errors.AsType[ExitStatus](tt.err)
			if ok != tt.wantOK {
				t.Fatalf("errors.AsType ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && uint8(es) != tt.want {
				t.Errorf("errors.AsType extracted status = %d, want %d", uint8(es), tt.want)
			}
		})
	}
}
