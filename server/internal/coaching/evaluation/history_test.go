package evaluation

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	in := HistoryCursor{CompletedAt: time.Date(2026, 9, 3, 8, 0, 0, 123, time.UTC), ID: "report-1"}
	encoded, err := EncodeCursor(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !out.CompletedAt.Equal(in.CompletedAt) || out.ID != in.ID {
		t.Fatalf("cursor mismatch: %#v", out)
	}
}
func TestCursorRejectsMalformed(t *testing.T) {
	for _, value := range []string{"", " ", "not-base64", "YQ"} {
		if _, err := DecodeCursor(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}
