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
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/go-gst/go-gst/gst"
	"github.com/go-gst/go-gst/gst/app"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"

	"github.com/livekit/protocol/logger"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

const minKeyframeRequestInterval = 500 * time.Millisecond

type publisherTrack struct {
	track                 *lksdk.LocalTrack
	sink                  *app.Sink
	mimeType              string
	publication           *lksdk.LocalTrackPublication
	isEnded               atomic.Bool
	onEOS                 func()
	lastKeyframeRequestNs atomic.Int64
	clockRate             uint32
	// only touched from the appsink's streaming thread
	rtp rtpClock
}

// rtpClock turns each sample's capture time (its GStreamer running time) into
// the RTP timestamp it is published with, so RTP timestamps follow real time.
// A sample without a usable capture time goes one duration on from the last.
type rtpClock struct {
	started bool
	last    uint32
}

// rtpTimestampBase keeps the first timestamps away from zero, which the SDK
// reads as "no timestamp given".
const rtpTimestampBase = 1 << 16

func rtpTicks(d time.Duration, clockRate uint32) uint32 {
	if d <= 0 {
		return 0
	}
	// Whole seconds first, so hours of running time don't overflow.
	sec := uint64(d / time.Second)
	rem := uint64(d % time.Second)
	return uint32(sec*uint64(clockRate) + rem*uint64(clockRate)/uint64(time.Second))
}

// next returns the RTP timestamp for a sample. `running` is its capture time
// and `valid` whether it has one.
//
// The timestamp is the capture time on the RTP clock. The SDK takes it as an
// absolute position (media.Sample.PacketTimestamp), so a sample that is held
// back stays the only one off: the next sample with a capture time is on real
// time again. It holds a sample back in two cases, and this does the same to
// stay in step with it:
//   - no capture time, or one that doesn't move forward (the SDK rejects
//     those, which would end the stream): one duration on from the last
//     sample, as a sample published by duration alone;
//   - less than its duration after the last sample: one duration on.
func (c *rtpClock) next(running time.Duration, valid bool, duration time.Duration, clockRate uint32) uint32 {
	step := rtpTicks(duration, clockRate)
	ts := c.last + step
	if valid {
		captured := rtpTimestampBase + rtpTicks(running, clockRate)
		// Signed, so it also holds when the 32-bit timestamp wraps around
		if !c.started || int32(captured-ts) > 0 {
			ts = captured
		}
	} else if !c.started {
		ts = rtpTimestampBase
	}
	if ts == 0 {
		// Zero means "not given" to the SDK. One tick late, once every 13 hours
		// of video.
		ts = 1
	}
	c.started = true
	c.last = ts
	return ts
}

func rtpClockRate(webrtcMime string) uint32 {
	if webrtcMime == webrtc.MimeTypeOpus {
		return 48000
	}
	return 90000
}

func createPublisherTrack(mimeType string, sync bool) (*publisherTrack, error) {
	webrtcMime := ""
	if mimeType == "audio/x-opus" {
		webrtcMime = webrtc.MimeTypeOpus
	} else if mimeType == "video/x-h264" {
		webrtcMime = webrtc.MimeTypeH264
	} else if mimeType == "video/x-vp8" {
		webrtcMime = webrtc.MimeTypeVP8
	} else if mimeType == "video/x-vp9" {
		webrtcMime = webrtc.MimeTypeVP9
	} else if mimeType == "video/x-av1" {
		webrtcMime = webrtc.MimeTypeAV1
	} else {
		return nil, fmt.Errorf("unsupported mime type: %v", mimeType)
	}

	sink, err := app.NewAppSink()
	if err != nil {
		return nil, err
	}

	if !sync {
		// With sync on, the sink holds each sample until its timestamp plus the
		// pipeline latency, which is the slowest branch's (audio: about 50 ms).
		if err := sink.Element.SetProperty("sync", false); err != nil {
			return nil, err
		}
	}

	pt := &publisherTrack{
		mimeType:  webrtcMime,
		sink:      sink,
		clockRate: rtpClockRate(webrtcMime),
	}
	sink.SetCallbacks(&app.SinkCallbacks{
		EOSFunc:       pt.handleEOS,
		NewSampleFunc: pt.handleSample,
	})

	if mimeType == "video/x-h264" {
		sink.SetCaps(gst.NewCapsFromString("video/x-h264,stream-format=byte-stream"))
	}

	pt.track, err = lksdk.NewLocalTrack(
		webrtc.RTPCodecCapability{MimeType: webrtcMime},
		lksdk.WithRTCPHandler(pt.onRTCP),
	)
	if err != nil {
		return nil, err
	}
	return pt, nil
}

func (t *publisherTrack) IsEnded() bool {
	return t.isEnded.Load()
}

// callback function when EOS is received
func (t *publisherTrack) handleEOS(_ *app.Sink) {
	t.isEnded.Store(true)
	if t.onEOS != nil {
		t.onEOS()
	}
}

// callback function when new sample is ready
func (t *publisherTrack) handleSample(sink *app.Sink) gst.FlowReturn {
	s := sink.PullSample()
	if s == nil {
		return gst.FlowEOS
	}

	buffer := s.GetBuffer()
	if buffer == nil {
		return gst.FlowError
	}

	segment := s.GetSegment()
	if segment == nil {
		return gst.FlowError
	}

	duration := buffer.Duration()

	// Publish with the capture time, so RTP timestamps follow real time. With
	// only a duration they advance one frame per sample, and every frame the
	// pipeline drops (a leaky queue, a busy encoder) leaves them further behind
	// the clock, which receivers answer with a longer jitter buffer.
	pts := buffer.PresentationTimestamp()
	running := gst.ClockTimeNone
	if pts != gst.ClockTimeNone {
		running = gst.ClockTime(segment.ToRunningTime(gst.FormatTime, uint64(pts)))
	}
	rtpTimestamp := t.rtp.next(time.Duration(running), running != gst.ClockTimeNone, time.Duration(duration), t.clockRate)

	err := t.track.WriteSample(media.Sample{
		Data:            buffer.Bytes(),
		Duration:        time.Duration(duration),
		PacketTimestamp: rtpTimestamp,
	}, nil)

	switch {
	case err == nil:
		return gst.FlowOK
	case errors.Is(err, io.EOF):
		return gst.FlowEOS
	default:
		return gst.FlowError
	}
}

func (t *publisherTrack) onRTCP(packet rtcp.Packet) {
	switch packet.(type) {
	case *rtcp.PictureLossIndication, *rtcp.FullIntraRequest:
		t.forceKeyframe()
	}
}

func (t *publisherTrack) forceKeyframe() {
	if t.mimeType == webrtc.MimeTypeOpus {
		return
	}

	now := time.Now().UnixNano()
	last := t.lastKeyframeRequestNs.Load()
	if now-last < int64(minKeyframeRequestInterval) {
		return
	}
	if !t.lastKeyframeRequestNs.CompareAndSwap(last, now) {
		return
	}

	s := gst.NewStructure("GstForceKeyUnit")
	_ = s.SetValue("all-headers", true)
	ev := gst.NewCustomEvent(gst.EventTypeCustomUpstream, s)

	pad := t.sink.GetStaticPad("sink")
	if pad == nil {
		return
	}

	// PushEvent forwards the event to the pad's peer. For the appsink's sink
	// pad, the peer is the src pad of the upstream element, so the upstream
	// force-key-unit event travels upstream toward the encoder. SendEvent
	// would be rejected by GStreamer as "wrong direction" on a sink pad.
	if ok := pad.PushEvent(ev); !ok {
		logger.Warnw("force-keyframe event was not handled by pipeline", nil)
	}
}
