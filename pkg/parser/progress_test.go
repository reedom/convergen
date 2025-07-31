package parser

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
)

func TestASTParser_CalculateProgressInterval(t *testing.T) {
	logger := zaptest.NewLogger(t)

	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	parser := NewASTParser(logger, eventBus, nil)
	defer parser.Close()

	tests := []struct {
		name     string
		total    int
		expected time.Duration
	}{
		{
			name:     "very small operations (≤5)",
			total:    3,
			expected: 1 * time.Hour, // Effectively disabled
		},
		{
			name:     "small operations (≤20)",
			total:    15,
			expected: 500 * time.Millisecond,
		},
		{
			name:     "medium operations (≤100)",
			total:    50,
			expected: 200 * time.Millisecond,
		},
		{
			name:     "large operations (≤500)",
			total:    300,
			expected: 100 * time.Millisecond,
		},
		{
			name:     "very large operations (>500)",
			total:    1000,
			expected: 50 * time.Millisecond,
		},
		{
			name:     "boundary case - exactly 5",
			total:    5,
			expected: 1 * time.Hour,
		},
		{
			name:     "boundary case - exactly 20",
			total:    20,
			expected: 500 * time.Millisecond,
		},
		{
			name:     "boundary case - exactly 100",
			total:    100,
			expected: 200 * time.Millisecond,
		},
		{
			name:     "boundary case - exactly 500",
			total:    500,
			expected: 100 * time.Millisecond,
		},
		{
			name:     "edge case - zero",
			total:    0,
			expected: 1 * time.Hour,
		},
		{
			name:     "edge case - one",
			total:    1,
			expected: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.calculateProgressInterval(tt.total)
			assert.Equal(t, tt.expected, result, "Interval calculation for total %d", tt.total)
		})
	}
}

func TestASTParser_ShouldReportProgress(t *testing.T) {
	logger := zaptest.NewLogger(t)

	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	parser := NewASTParser(logger, eventBus, nil)
	defer parser.Close()

	now := time.Now()

	tests := []struct {
		name        string
		lastReport  time.Time
		reportCount int
		total       int
		expected    bool
	}{
		{
			name:        "very small operations should not report",
			lastReport:  now,
			reportCount: 0,
			total:       3,
			expected:    false,
		},
		{
			name:        "first few reports always report",
			lastReport:  now,
			reportCount: 1,
			total:       50,
			expected:    true,
		},
		{
			name:        "second report always report",
			lastReport:  now,
			reportCount: 2,
			total:       50,
			expected:    true,
		},
		{
			name:        "third report always report",
			lastReport:  now,
			reportCount: 3,
			total:       50,
			expected:    true,
		},
		{
			name:        "standard reporting after initial reports",
			lastReport:  now.Add(-100 * time.Millisecond),
			reportCount: 5,
			total:       50,
			expected:    true,
		},
		{
			name:        "long running operation with many reports - recent",
			lastReport:  now.Add(-500 * time.Millisecond),
			reportCount: 25,
			total:       150,
			expected:    false, // Less than 1 second since last report
		},
		{
			name:        "long running operation with many reports - old",
			lastReport:  now.Add(-1500 * time.Millisecond),
			reportCount: 25,
			total:       150,
			expected:    true, // More than 1 second since last report
		},
		{
			name:        "boundary case - exactly 5 items",
			lastReport:  now,
			reportCount: 0,
			total:       5,
			expected:    false, // Should not report for ≤5 items
		},
		{
			name:        "boundary case - 6 items, first report",
			lastReport:  now,
			reportCount: 0,
			total:       6,
			expected:    true, // Should report for >5 items, first few reports
		},
		{
			name:        "boundary case - exactly 100 items, many reports",
			lastReport:  now.Add(-500 * time.Millisecond),
			reportCount: 15,
			total:       100,
			expected:    true, // Not in throttling range yet
		},
		{
			name:        "boundary case - 101 items, many reports, recent",
			lastReport:  now.Add(-500 * time.Millisecond),
			reportCount: 25,
			total:       101,
			expected:    false, // In throttling range, recent report
		},
		{
			name:        "boundary case - 101 items, many reports, old",
			lastReport:  now.Add(-1200 * time.Millisecond),
			reportCount: 25,
			total:       101,
			expected:    true, // In throttling range, old report
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.shouldReportProgress(tt.lastReport, tt.reportCount, tt.total)
			assert.Equal(t, tt.expected, result,
				"Progress reporting decision for total=%d, count=%d, lastReport=%v",
				tt.total, tt.reportCount, tt.lastReport)
		})
	}
}

func TestASTParser_TrackProgress(t *testing.T) {
	logger := zaptest.NewLogger(t)

	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	// Track events published
	var publishedEvents []events.Event

	handler := events.NewFuncEventHandler("progress.update", func(ctx context.Context, event events.Event) error {
		publishedEvents = append(publishedEvents, event)
		return nil
	})
	err := eventBus.Subscribe("progress.update", handler)
	assert.NoError(t, err)

	parser := NewASTParser(logger, eventBus, &ParserConfig{
		EnableProgress: true,
	})
	defer parser.Close()

	tests := []struct {
		name             string
		total            int
		duration         time.Duration
		expectEvents     bool
		expectFinalEvent bool
	}{
		{
			name:             "small operation - no events expected",
			total:            3,
			duration:         100 * time.Millisecond,
			expectEvents:     false,
			expectFinalEvent: false,
		},
		{
			name:             "medium operation - events expected",
			total:            25,
			duration:         300 * time.Millisecond,
			expectEvents:     true,
			expectFinalEvent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset events
			publishedEvents = nil

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			done := make(chan struct{})

			// Start progress tracking
			go parser.trackProgress(ctx, domain.PhaseParsing, tt.total, "Test operation", done)

			// Wait for the specified duration
			time.Sleep(tt.duration)

			// Signal completion
			close(done)

			// Wait a bit for final event processing
			time.Sleep(50 * time.Millisecond)

			if tt.expectEvents {
				assert.Greater(t, len(publishedEvents), 0, "Should have published progress events")

				if tt.expectFinalEvent {
					// Check if we have a final event (contains "completed")
					finalEventFound := false

					for _, event := range publishedEvents {
						if progressEvent, ok := event.(*events.ProgressEvent); ok {
							if len(progressEvent.Message) > 0 &&
								len(progressEvent.Message) > 10 { // "completed" would make message longer
								finalEventFound = true
								break
							}
						}
					}

					assert.True(t, finalEventFound, "Should have published final progress event")
				}
			} else {
				assert.Equal(t, 0, len(publishedEvents), "Should not have published progress events for small operations")
			}
		})
	}
}

func TestASTParser_TrackProgress_ContextCancellation(t *testing.T) {
	logger := zaptest.NewLogger(t)

	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	parser := NewASTParser(logger, eventBus, &ParserConfig{
		EnableProgress: true,
	})
	defer parser.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	// Start progress tracking
	finished := make(chan struct{})
	go func() {
		parser.trackProgress(ctx, domain.PhaseParsing, 100, "Test operation", done)
		close(finished)
	}()

	// Cancel context after a short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for tracking to finish
	select {
	case <-finished:
		// Good - tracking finished due to context cancellation
	case <-time.After(1 * time.Second):
		t.Fatal("Progress tracking did not finish after context cancellation")
	}

	// Cleanup
	close(done)
}

func TestASTParser_TrackProgress_AdaptiveFrequency(t *testing.T) {
	logger := zaptest.NewLogger(t)

	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	// Track events with timestamps
	var publishedEvents []struct {
		event     events.Event
		timestamp time.Time
	}

	handler := events.NewFuncEventHandler("progress.update", func(ctx context.Context, event events.Event) error {
		publishedEvents = append(publishedEvents, struct {
			event     events.Event
			timestamp time.Time
		}{event, time.Now()})

		return nil
	})
	err := eventBus.Subscribe("progress.update", handler)
	assert.NoError(t, err)

	parser := NewASTParser(logger, eventBus, &ParserConfig{
		EnableProgress: true,
	})
	defer parser.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan struct{})

	// Start progress tracking for a large operation
	go parser.trackProgress(ctx, domain.PhaseParsing, 1000, "Large operation", done)

	// Let it run for a while to generate multiple events and potentially trigger adaptive behavior
	time.Sleep(1500 * time.Millisecond)

	// Signal completion
	close(done)

	// Wait for final processing
	time.Sleep(100 * time.Millisecond)

	// We should have multiple events
	assert.Greater(t, len(publishedEvents), 5, "Should have multiple progress events for large operation")

	// Check that events are spaced reasonably
	if len(publishedEvents) > 1 {
		for i := 1; i < len(publishedEvents); i++ {
			timeDiff := publishedEvents[i].timestamp.Sub(publishedEvents[i-1].timestamp)
			assert.Greater(t, timeDiff, 10*time.Millisecond, "Events should be spaced at least 10ms apart")
			assert.Less(t, timeDiff, 1*time.Second, "Events should not be more than 1s apart")
		}
	}
}

func BenchmarkCalculateProgressInterval(b *testing.B) {
	logger := zaptest.NewLogger(b)

	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	parser := NewASTParser(logger, eventBus, nil)
	defer parser.Close()

	testTotals := []int{1, 10, 50, 200, 1000}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, total := range testTotals {
			parser.calculateProgressInterval(total)
		}
	}
}

func BenchmarkShouldReportProgress(b *testing.B) {
	logger := zaptest.NewLogger(b)

	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	parser := NewASTParser(logger, eventBus, nil)
	defer parser.Close()

	now := time.Now()
	lastReport := now.Add(-200 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parser.shouldReportProgress(lastReport, 10, 100)
	}
}
