package payment

import (
	"testing"
	"time"
)

func fixedClock() Clock {
	t := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func mustMoney(t *testing.T, cents int64) Money {
	t.Helper()
	m, err := NewMoney(cents, BRL)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	return m
}

func newCaptured(t *testing.T, cents int64) *Payment {
	t.Helper()
	clk := fixedClock()
	p, err := NewPayment("pay_1", "idem-key-123", "cust_1", mustMoney(t, cents), clk)
	if err != nil {
		t.Fatalf("NewPayment: %v", err)
	}
	if err := p.Authorize("ref_1", clk); err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if err := p.Capture(clk); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	return p
}

func TestLifecycleHappyPath(t *testing.T) {
	p := newCaptured(t, 10_000)
	if p.Status() != StatusCaptured {
		t.Fatalf("status = %s, want CAPTURED", p.Status())
	}
}

func TestCannotCaptureBeforeAuthorize(t *testing.T) {
	clk := fixedClock()
	p, _ := NewPayment("pay_2", "idem-key-456", "cust_1", mustMoney(t, 5000), clk)
	if err := p.Capture(clk); err != ErrNotAuthorized {
		t.Fatalf("err = %v, want ErrNotAuthorized", err)
	}
}

func TestPartialThenFullRefund(t *testing.T) {
	clk := fixedClock()
	p := newCaptured(t, 10_000)

	if err := p.Refund(mustMoney(t, 4_000), clk); err != nil {
		t.Fatalf("partial refund: %v", err)
	}
	if p.Status() != StatusPartiallyRefund {
		t.Fatalf("status = %s, want PARTIALLY_REFUNDED", p.Status())
	}

	if err := p.Refund(mustMoney(t, 6_000), clk); err != nil {
		t.Fatalf("final refund: %v", err)
	}
	if p.Status() != StatusRefunded {
		t.Fatalf("status = %s, want REFUNDED", p.Status())
	}
}

func TestRefundCannotExceedCaptured(t *testing.T) {
	clk := fixedClock()
	p := newCaptured(t, 10_000)
	if err := p.Refund(mustMoney(t, 12_000), clk); err != ErrRefundExceedsTotal {
		t.Fatalf("err = %v, want ErrRefundExceedsTotal", err)
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	p := newCaptured(t, 10_000)
	got := FromSnapshot(p.ToSnapshot())
	if got.Status() != p.Status() || got.Amount().AmountCents() != p.Amount().AmountCents() {
		t.Fatalf("round-trip mismatch")
	}
}
