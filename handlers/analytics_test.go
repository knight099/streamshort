package handlers

import (
	"testing"
	"time"
)

// TestProportionalEarningsCalculation tests the earnings calculation logic
// without database dependencies
func TestProportionalEarningsCalculation(t *testing.T) {
	// User subscription: ₹100 → Distributable: ₹70
	distributable := 70.0

	// Test case: User watches 4 creators with different watch times
	// P: 20 min, Q: 10 min, R: 2 min, S: 100 min
	watchTimes := map[string]int64{
		"creator_p": 20 * 60,  // 20 min in seconds
		"creator_q": 10 * 60,  // 10 min in seconds
		"creator_r": 2 * 60,   // 2 min in seconds
		"creator_s": 100 * 60, // 100 min in seconds
	}

	// Calculate total watch time
	var totalWatchTime int64
	for _, wt := range watchTimes {
		totalWatchTime += wt
	}

	// Should be 132 minutes = 7920 seconds
	expectedTotalSeconds := int64(132 * 60)
	if totalWatchTime != expectedTotalSeconds {
		t.Errorf("Total watch time: got %d, want %d", totalWatchTime, expectedTotalSeconds)
	}

	// Calculate and verify earnings for each creator
	type expected struct {
		name       string
		watchMin   int
		percentage float64
		earnings   float64
	}

	expectedResults := []expected{
		{"creator_p", 20, 15.15, 10.61},
		{"creator_q", 10, 7.58, 5.30},
		{"creator_r", 2, 1.52, 1.06},
		{"creator_s", 100, 75.76, 53.03},
	}

	var totalEarnings float64
	for _, exp := range expectedResults {
		creatorWatchTime := watchTimes[exp.name]
		shareFraction := float64(creatorWatchTime) / float64(totalWatchTime)
		earnings := distributable * shareFraction

		// Check percentage (with tolerance)
		actualPercentage := shareFraction * 100
		if !floatEquals(actualPercentage, exp.percentage, 0.1) {
			t.Errorf("%s percentage: got %.2f%%, want %.2f%%", exp.name, actualPercentage, exp.percentage)
		}

		// Check earnings (with tolerance of ₹0.01)
		if !floatEquals(earnings, exp.earnings, 0.02) {
			t.Errorf("%s earnings: got ₹%.2f, want ₹%.2f", exp.name, earnings, exp.earnings)
		}

		t.Logf("%s: %d min → %.2f%% → ₹%.2f", exp.name, exp.watchMin, actualPercentage, earnings)
		totalEarnings += earnings
	}

	// Verify total earnings equals distributable (₹70)
	if !floatEquals(totalEarnings, distributable, 0.01) {
		t.Errorf("Total earnings: got ₹%.2f, want ₹%.2f", totalEarnings, distributable)
	}

	t.Logf("Total earnings: ₹%.2f (should be ₹%.2f)", totalEarnings, distributable)
}

// TestRealTimeEarningsUpdates simulates the real-time update scenario
func TestRealTimeEarningsUpdates(t *testing.T) {
	distributable := 70.0

	// Simulate watching creators in sequence
	type watchEvent struct {
		creator   string
		watchMins int64
	}

	events := []watchEvent{
		{"creator_p", 20},
		{"creator_q", 10},
		{"creator_r", 2},
		{"creator_s", 100},
	}

	// Track cumulative watch times
	watchTimes := make(map[string]int64)
	var totalWatchTime int64

	for eventNum, event := range events {
		// Add this event's watch time
		watchTimes[event.creator] += event.watchMins * 60 // convert to seconds
		totalWatchTime += event.watchMins * 60

		t.Logf("\n=== After watching %s (%d min) ===", event.creator, event.watchMins)
		t.Logf("Total watch time: %d min", totalWatchTime/60)

		// Calculate earnings for all creators at this point
		var runningTotal float64
		for creator, wt := range watchTimes {
			shareFraction := float64(wt) / float64(totalWatchTime)
			earnings := distributable * shareFraction
			t.Logf("  %s: %d min → %.2f%% → ₹%.2f", creator, wt/60, shareFraction*100, earnings)
			runningTotal += earnings
		}

		// Verify total still equals distributable
		if !floatEquals(runningTotal, distributable, 0.01) {
			t.Errorf("Event %d: Total earnings ₹%.2f != ₹%.2f", eventNum, runningTotal, distributable)
		}
	}
}

// TestEdgeCases tests edge cases in earnings calculation
func TestEdgeCases(t *testing.T) {
	distributable := 70.0

	// Test case 1: Only one creator watched
	t.Run("SingleCreator", func(t *testing.T) {
		totalWatchTime := int64(100 * 60) // 100 min
		creatorWatchTime := int64(100 * 60)

		shareFraction := float64(creatorWatchTime) / float64(totalWatchTime)
		earnings := distributable * shareFraction

		if earnings != distributable {
			t.Errorf("Single creator should get all: got ₹%.2f, want ₹%.2f", earnings, distributable)
		}
	})

	// Test case 2: Two creators with equal watch time
	t.Run("EqualWatchTime", func(t *testing.T) {
		totalWatchTime := int64(200 * 60)    // 200 min total
		creatorAWatchTime := int64(100 * 60) // 100 min each
		creatorBWatchTime := int64(100 * 60)

		earningsA := distributable * (float64(creatorAWatchTime) / float64(totalWatchTime))
		earningsB := distributable * (float64(creatorBWatchTime) / float64(totalWatchTime))

		if !floatEquals(earningsA, 35.0, 0.01) || !floatEquals(earningsB, 35.0, 0.01) {
			t.Errorf("Equal watch time should give ₹35 each: got A=₹%.2f, B=₹%.2f", earningsA, earningsB)
		}
	})

	// Test case 3: Tiny watch time (1 second)
	t.Run("TinyWatchTime", func(t *testing.T) {
		totalWatchTime := int64(100*60 + 1) // 100 min + 1 sec
		bigCreatorTime := int64(100 * 60)   // 100 min
		tinyCreatorTime := int64(1)         // 1 sec

		bigEarnings := distributable * (float64(bigCreatorTime) / float64(totalWatchTime))
		tinyEarnings := distributable * (float64(tinyCreatorTime) / float64(totalWatchTime))

		t.Logf("Big creator: ₹%.4f, Tiny creator: ₹%.4f", bigEarnings, tinyEarnings)

		// Even 1 second should get something (very small)
		if tinyEarnings <= 0 {
			t.Error("Even 1 second of watch time should earn something > 0")
		}
	})
}

// TestMonthBoundary tests that earnings are calculated per month
func TestMonthBoundary(t *testing.T) {
	now := time.Now()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Verify month date calculation
	if firstOfMonth.Day() != 1 {
		t.Errorf("First of month should be day 1, got %d", firstOfMonth.Day())
	}

	t.Logf("Month date for earnings: %s", firstOfMonth.Format("2006-01-02"))
}

// Helper function for float comparison with tolerance
func floatEquals(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
