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
	"io"
	"testing"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"

	lksdk "github.com/livekit/server-sdk-go/v2"
)

// A stand-in for the peer connection a track is bound to. It keeps the RTP
// timestamp of every packet the SDK sends.
type recordingContext struct {
	timestamps []uint32
}

func (c *recordingContext) CodecParameters() []webrtc.RTPCodecParameters {
	return []webrtc.RTPCodecParameters{{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000},
		PayloadType:        96,
	}}
}
func (c *recordingContext) HeaderExtensions() []webrtc.RTPHeaderExtensionParameter { return nil }
func (c *recordingContext) SSRC() webrtc.SSRC                                      { return 1 }
func (c *recordingContext) SSRCRetransmission() webrtc.SSRC                        { return 0 }
func (c *recordingContext) SSRCForwardErrorCorrection() webrtc.SSRC                { return 0 }
func (c *recordingContext) WriteStream() webrtc.TrackLocalWriter                   { return c }
func (c *recordingContext) ID() string                                             { return "test" }
func (c *recordingContext) RTCPReader() interceptor.RTCPReader {
	return interceptor.RTCPReaderFunc(func([]byte, interceptor.Attributes) (int, interceptor.Attributes, error) {
		return 0, nil, io.EOF
	})
}
func (c *recordingContext) WriteRTP(header *rtp.Header, _ []byte) (int, error) {
	c.timestamps = append(c.timestamps, header.Timestamp)
	return 0, nil
}
func (c *recordingContext) Write(b []byte) (int, error) { return len(b), nil }

type timedSample struct {
	// capture time in ms; negative means the buffer has no timestamp
	running  int
	duration int
}

// Publishes the samples the way handleSample does, through the SDK's own
// packetizer, and returns each one's RTP time in ms from the first.
func publishedTimes(t *testing.T, samples []timedSample) []int {
	t.Helper()
	track, err := lksdk.NewLocalTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &recordingContext{}
	if _, err := track.Bind(ctx); err != nil {
		t.Fatal(err)
	}
	var clock rtpClock
	var times []int
	for i, s := range samples {
		before := len(ctx.timestamps)
		ts := clock.next(time.Duration(s.running)*time.Millisecond, s.running >= 0, time.Duration(s.duration)*time.Millisecond, 90000)
		err := track.WriteSample(media.Sample{
			// A VP8 frame small enough for one packet
			Data:            []byte{0x10, 0x02, 0x00, 0x9d, 0x01, 0x2a},
			Duration:        sdkSampleDuration(90000),
			PacketTimestamp: ts,
		}, nil)
		if err != nil {
			t.Fatalf("sample %d: %v", i, err)
		}
		if len(ctx.timestamps) == before {
			t.Fatalf("sample %d: nothing was sent", i)
		}
		times = append(times, int(int32(ctx.timestamps[before]-ctx.timestamps[0]))/90)
	}
	return times
}

func TestPublishedTimestampsFollowCaptureTime(t *testing.T) {
	cases := []struct {
		name    string
		samples []timedSample
		want    []int
	}{
		{
			"steady frames",
			[]timedSample{{1000, 40}, {1040, 40}, {1080, 40}},
			[]int{0, 40, 80},
		},
		{
			// A frame dropped upstream leaves a gap. Published by duration
			// alone the timestamps would be 0, 40, 80, 120.
			"a dropped frame keeps its gap",
			[]timedSample{{0, 40}, {40, 40}, {120, 40}, {160, 40}},
			[]int{0, 40, 120, 160},
		},
		{
			"durations that differ from frame to frame",
			[]timedSample{{0, 20}, {20, 40}, {60, 20}, {80, 40}, {120, 20}, {140, 40}, {180, 20}},
			[]int{0, 20, 60, 80, 120, 140, 180},
		},
		{
			"a frame captured right after the last one is published right after it",
			[]timedSample{{0, 40}, {40, 40}, {45, 40}, {120, 40}, {160, 40}},
			[]int{0, 40, 45, 120, 160},
		},
		{
			// Captured late, then on time again: published a whole duration
			// apart, every frame after the late one would be 39 ms ahead.
			"a late frame doesn't push the ones after it ahead",
			[]timedSample{{0, 40}, {40, 40}, {41, 40}, {81, 40}, {121, 40}},
			[]int{0, 40, 41, 81, 121},
		},
		{
			"a gap between two samples of different durations is kept as it is",
			[]timedSample{{0, 20}, {100, 40}, {140, 40}},
			[]int{0, 100, 140},
		},
		{
			"a long sample before a short one",
			[]timedSample{{0, 40}, {40, 20}, {60, 40}, {100, 20}},
			[]int{0, 40, 60, 100},
		},
		{
			"a buffer without a timestamp before one of another duration",
			[]timedSample{{0, 40}, {40, 40}, {-1, 40}, {200, 20}},
			[]int{0, 40, 80, 200},
		},
		{
			"a buffer without a timestamp goes one duration on, and the gap after it is kept",
			[]timedSample{{0, 40}, {40, 40}, {-1, 40}, {200, 40}},
			[]int{0, 40, 80, 200},
		},
		{
			"a timestamp that goes back a little is published right after the last, and the gap after it is kept",
			[]timedSample{{0, 40}, {40, 40}, {20, 40}, {160, 40}},
			[]int{0, 40, 40, 160},
		},
		{
			// Published a duration apart from there on, every frame after
			// the missing timestamp would be 40 ms ahead for good.
			"the frames after a buffer without a timestamp are on capture time again",
			[]timedSample{{0, 40}, {40, 40}, {-1, 40}, {80, 40}, {120, 40}, {160, 40}},
			[]int{0, 40, 80, 80, 120, 160},
		},
		{
			"and so are the frames after a timestamp that went back a little",
			[]timedSample{{0, 40}, {40, 40}, {20, 40}, {60, 40}, {100, 40}},
			[]int{0, 40, 40, 60, 100},
		},
		{
			// Capture times that start over, and go on past where they
			// were before. A tick apart, the frames would all be shown at
			// once.
			"timestamps that start over keep their spacing from there",
			[]timedSample{{5000, 1000}, {6000, 1000}, {0, 1000}, {1000, 1000}, {3000, 1000}, {4000, 1000}, {5000, 1000}, {6000, 1000}, {7000, 1000}, {8000, 1000}},
			[]int{0, 1000, 1000, 2000, 4000, 5000, 6000, 7000, 8000, 9000},
		},
		{
			// The first frame after the restart goes a tick after the last
			// one, and the frame dropped after it keeps its gap.
			"timestamps that start over a moment after the start",
			[]timedSample{{600, 40}, {640, 40}, {0, 40}, {40, 40}, {120, 40}, {160, 40}},
			[]int{0, 40, 40, 80, 160, 200},
		},
		{
			"and with nothing dropped",
			[]timedSample{{800, 40}, {840, 40}, {880, 40}, {0, 40}, {40, 40}, {80, 40}},
			[]int{0, 40, 80, 80, 120, 160},
		},
		{
			"nor does one that is followed by a buffer without a timestamp",
			[]timedSample{{600, 40}, {640, 40}, {0, 40}, {-1, 40}, {680, 40}, {720, 40}},
			[]int{0, 40, 40, 80, 80, 120},
		},
		{
			"nor one that comes twice",
			[]timedSample{{5000, 40}, {5040, 40}, {0, 40}, {0, 40}, {5080, 40}},
			[]int{0, 40, 40, 40, 80},
		},
		{
			// The buffer without a timestamp goes 120 ms on, past the two
			// after it. Their capture times never went back: they aren't
			// taken for a restart.
			"a long buffer without a timestamp isn't taken for a restart",
			[]timedSample{{0, 40}, {40, 40}, {-1, 120}, {80, 40}, {120, 40}, {160, 40}, {200, 40}},
			[]int{0, 40, 160, 160, 160, 160, 200},
		},
		{
			"one stray timestamp doesn't move the frames after it",
			[]timedSample{{5000, 40}, {5040, 40}, {0, 40}, {5080, 40}, {5120, 40}},
			[]int{0, 40, 40, 80, 120},
		},
		{
			"a first buffer without a timestamp",
			[]timedSample{{-1, 40}, {5000, 40}, {5040, 40}},
			[]int{0, 5000, 5040},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := publishedTimes(t, tc.samples)
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("RTP times %v ms, want %v ms", got, tc.want)
				}
			}
		})
	}
}

func TestRTPClockCrossesTheTimestampWrap(t *testing.T) {
	// One tick a millisecond, so the times below are whole ticks. The clock
	// starts rtpTimestampBase in, so it wraps that much before 2^32 ticks of
	// running time.
	const rate = 1000
	wrap := time.Duration(1<<32-rtpTimestampBase) * time.Millisecond
	var clock rtpClock
	before := clock.next(wrap-20*time.Millisecond, true, 20*time.Millisecond, rate)
	at := clock.next(wrap, true, 20*time.Millisecond, rate)
	after := clock.next(wrap+20*time.Millisecond, true, 20*time.Millisecond, rate)
	if before != 1<<32-20 {
		t.Fatalf("20 ms before the wrap is timestamp %d, want %d", before, uint32(1<<32-20))
	}
	// Exactly on the wrap the timestamp would be zero, which the SDK reads
	// as "not given"
	if at != 1 {
		t.Fatalf("the timestamp on the wrap is %d, want 1", at)
	}
	if after != 20 {
		t.Fatalf("20 ms after the wrap is timestamp %d, want 20", after)
	}
}

func TestRTPClockRates(t *testing.T) {
	if got := rtpClockRate(webrtc.MimeTypeOpus); got != 48000 {
		t.Errorf("Opus clock rate = %d, want 48000", got)
	}
	if got := rtpClockRate(webrtc.MimeTypeVP8); got != 90000 {
		t.Errorf("VP8 clock rate = %d, want 90000", got)
	}
	if got := rtpTicks(20*time.Millisecond, 48000); got != 960 {
		t.Errorf("20 ms of Opus = %d ticks, want 960", got)
	}
	// The way the SDK counts a sample's duration in ticks
	for _, rate := range []uint32{48000, 90000} {
		if got := uint32(sdkSampleDuration(rate).Seconds() * float64(rate)); got != 1 {
			t.Errorf("the SDK reads the duration at %d Hz as %d ticks, want 1", rate, got)
		}
	}
}
