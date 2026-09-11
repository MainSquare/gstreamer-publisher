# LiveKit GStreamer Publisher

This utility allows you to publish from any GStreamer source to LiveKit.
It parses a gst-launch style pipeline and reads negotiates 

## Prerequisites

- Go 1.22 or later
- GStreamer 1.20 or later

## Install

```bash
go install github.com/MainSquare/gstreamer-publisher@latest
```

## Usage

To use this utility, you need to [generate an access token](https://docs.livekit.io/home/get-started/authentication/) that includes
the `canPublish` permission.

When constructing pipelines, you would want to end the pipeline with elements that produce H264, VP8, VP9, or Opus.
Do not mux the streams into a container format. GStreamer-publisher will inspect the pipeline and import the raw
streams into LiveKit.

### Subscription allowlist

Pass `--allowed-users` with a comma-separated list of LiveKit participant identities to restrict who can subscribe
to the published tracks:

- On startup only the listed identities are allowed to subscribe; everyone else is denied.
- While the publisher runs, it reads updated allowlists from stdin: each line is a full, comma-separated list and
  is applied when the process receives `SIGUSR1`.
- An empty (or otherwise identity-free) line is an explicit deny-all: no participant may subscribe, and participants
  that are currently subscribed are revoked.

Without `--allowed-users` the publisher does not read stdin at all and every participant in the room can subscribe.

## Examples

The examples below assume you have the following environment variables defined:

```bash
export LIVEKIT_URL=<wss://project.livekit.cloud>
export LIVEKIT_PUBLISH_TOKEN=<token>
```

To view the stream, you can create a second token and head over to [our example app](https://meet.livekit.io/?tab=custom) to connect as a viewer.

### Test stream as H.264

This creates a video and audio from test src, encoding it to H.264 and Opus.

```bash
gstreamer-publisher --token $LIVEKIT_PUBLISH_TOKEN \
    -- \
    videotestsrc is-live=true ! \
        video/x-raw,width=1280,height=720 ! \
        clockoverlay ! \
        videoconvert ! \
        x264enc tune=zerolatency key-int-max=60 bitrate=2000 \
    audiotestsrc is-live=true ! \
        audioresample ! \
        audioconvert ! \
        opusenc bitrate=64000
```

### Publish from file

The following converts any video file into VP9 and Opus using decodebin3

```bash
gstreamer-publisher --token $LIVEKIT_PUBLISH_TOKEN \
  -- \
    filesrc location="/path/to/file" ! \
    decodebin3 name=decoder \
    decoder. ! queue ! \
        videoconvert ! \
        videoscale ! \
        video/x-raw,width=1280,height=720 ! \
        vp9enc deadline=1 cpu-used=-6 row-mt=1 tile-columns=3 tile-rows=1 target-bitrate=2000000 keyframe-max-dist=60 \
    decoder. ! queue ! \
        audioconvert ! \
        audioresample ! \
        opusenc bitrate=64000
```
