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
	"reflect"
	"testing"

	"github.com/livekit/protocol/livekit"
)

func TestParseAllowedUsers(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty flag means feature off", "", nil},
		{"simple list", "alice,bob", []string{"alice", "bob"}},
		{"spaces are trimmed", " alice , bob ", []string{"alice", "bob"}},
		{"empty components skipped", ",alice,,bob,", []string{"alice", "bob"}},
		{"whitespace only", "   ", nil},
		{"commas only", ",,", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseAllowedUsers(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("parseAllowedUsers(%q) = %v, want %v", tc.input, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("parseAllowedUsers(%q) = %v, want %v", tc.input, got, tc.want)
				}
			}
		})
	}
}

func TestSubscriptionPermission(t *testing.T) {
	t.Run("empty user list is explicit deny-all", func(t *testing.T) {
		perm := subscriptionPermission(nil)
		if perm.AllParticipants {
			t.Fatal("AllParticipants should be false")
		}
		if len(perm.TrackPermissions) != 0 {
			t.Fatalf("TrackPermissions should be empty, got %v", perm.TrackPermissions)
		}
		// the value sent must be equivalent to &livekit.SubscriptionPermission{}
		if !reflect.DeepEqual(perm, &livekit.SubscriptionPermission{}) {
			t.Fatalf("got %+v, want zero-value SubscriptionPermission", perm)
		}
	})

	t.Run("identities become all-tracks permissions", func(t *testing.T) {
		perm := subscriptionPermission([]string{"alice", "bob"})
		if perm.AllParticipants {
			t.Fatal("AllParticipants should be false")
		}
		want := []*livekit.TrackPermission{
			{ParticipantIdentity: "alice", AllTracks: true},
			{ParticipantIdentity: "bob", AllTracks: true},
		}
		if !reflect.DeepEqual(perm.TrackPermissions, want) {
			t.Fatalf("got %+v, want %+v", perm.TrackPermissions, want)
		}
	})
}
