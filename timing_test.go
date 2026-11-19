// Copyright 2024 LiveKit, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"testing"
	"time"
)

func TestCaptureClockKeepsRunningTimeSpacing(t *testing.T) {
	clock := &captureClock{}
	first := clock.wallTime(2 * time.Second)
	if since := time.Since(first); since < 0 || since > time.Second {
		t.Fatalf("first sample should be anchored to now, got %v ago", since)
	}
	// A frame dropped upstream leaves a gap in running time. The gap has to
	// survive, or RTP timestamps fall behind the clock.
	if got := clock.wallTime(2*time.Second + 120*time.Millisecond).Sub(first); got != 120*time.Millisecond {
		t.Fatalf("gap = %v, want 120ms", got)
	}
	// A second track that starts later lands on the same timeline.
	if got := clock.wallTime(2*time.Second + 30*time.Millisecond).Sub(first); got != 30*time.Millisecond {
		t.Fatalf("other track offset = %v, want 30ms", got)
	}
}

func TestNextSampleTimeFallsBackToDuration(t *testing.T) {
	clock := &captureClock{}
	first := nextSampleTime(clock, time.Time{}, time.Second, true)
	if first.IsZero() {
		t.Fatal("a timestamped sample should get a capture time")
	}
	cases := []struct {
		name    string
		clock   *captureClock
		running time.Duration
		valid   bool
	}{
		{"buffer without a timestamp", clock, 0, false},
		{"same time as the previous sample", clock, time.Second, true},
		{"earlier than the previous sample", clock, 900 * time.Millisecond, true},
		{"no clock", nil, 2 * time.Second, true},
	}
	for _, tc := range cases {
		if got := nextSampleTime(tc.clock, first, tc.running, tc.valid); !got.IsZero() {
			t.Errorf("%s: got %v, want the zero time", tc.name, got)
		}
	}
	if next := nextSampleTime(clock, first, time.Second+40*time.Millisecond, true); next.Sub(first) != 40*time.Millisecond {
		t.Errorf("next sample = %v after the first, want 40ms", next.Sub(first))
	}
}
