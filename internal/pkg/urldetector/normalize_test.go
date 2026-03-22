package urldetector

import (
	"testing"
)

func TestCanonicalizeURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// YouTube canonicalization
		{
			name:  "YouTube watch URL with tracking params",
			input: "https://www.youtube.com/watch?v=dQw4w9WgXcQ&si=abc123&feature=share",
			want:  "https://youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:  "YouTube short URL",
			input: "https://youtu.be/dQw4w9WgXcQ",
			want:  "https://youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:  "YouTube shorts URL",
			input: "https://youtube.com/shorts/dQw4w9WgXcQ",
			want:  "https://youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:  "YouTube embed URL",
			input: "https://youtube.com/embed/dQw4w9WgXcQ",
			want:  "https://youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:  "YouTube mobile URL",
			input: "https://m.youtube.com/watch?v=dQw4w9WgXcQ",
			want:  "https://youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:  "YouTube music URL",
			input: "https://music.youtube.com/watch?v=dQw4w9WgXcQ",
			want:  "https://youtube.com/watch?v=dQw4w9WgXcQ",
		},

		// Spotify canonicalization
		{
			name:  "Spotify track with tracking params",
			input: "https://open.spotify.com/track/4PTG3Z6ehGkBFwjybzWkR8?si=abc&pt=def&pi=ghi",
			want:  "https://open.spotify.com/track/4PTG3Z6ehGkBFwjybzWkR8",
		},
		{
			name:  "Spotify intl prefix",
			input: "https://open.spotify.com/intl-us/track/4PTG3Z6ehGkBFwjybzWkR8?si=test",
			want:  "https://open.spotify.com/track/4PTG3Z6ehGkBFwjybzWkR8",
		},
		{
			name:  "Spotify album",
			input: "https://open.spotify.com/album/4PTG3Z6ehGkBFwjybzWkR8",
			want:  "https://open.spotify.com/album/4PTG3Z6ehGkBFwjybzWkR8",
		},
		{
			name:  "Spotify playlist",
			input: "https://open.spotify.com/playlist/4PTG3Z6ehGkBFwjybzWkR8",
			want:  "https://open.spotify.com/playlist/4PTG3Z6ehGkBFwjybzWkR8",
		},

		// SoundCloud (no platform-specific extraction, just normalization)
		{
			name:  "SoundCloud with tracking params",
			input: "https://soundcloud.com/artist/track?si=abc&utm_source=twitter",
			want:  "https://soundcloud.com/artist/track",
		},
		{
			name:  "SoundCloud mobile domain",
			input: "https://m.soundcloud.com/artist/track",
			want:  "https://soundcloud.com/artist/track",
		},

		// General normalization
		{
			name:  "Strip www",
			input: "https://www.mixcloud.com/show/episode",
			want:  "https://mixcloud.com/show/episode",
		},
		{
			name:  "Remove fragment",
			input: "https://bandcamp.com/track#comment-123",
			want:  "https://bandcamp.com/track",
		},
		{
			name:  "Strip all tracking params",
			input: "https://example.com/path?keep=1&utm_source=x&fbclid=y&gclid=z&si=a&pt=b&pi=c&nd=d&feature=e&pp=f&ab_channel=g",
			want:  "https://example.com/path?keep=1",
		},
		{
			name:  "Sort query params",
			input: "https://example.com/path?z=3&a=1&m=2",
			want:  "https://example.com/path?a=1&m=2&z=3",
		},

		// Edge cases
		{
			name:    "Empty URL",
			input:   "",
			wantErr: true,
		},
		{
			name:  "URL without query params",
			input: "https://nts.live/shows/some-show",
			want:  "https://nts.live/shows/some-show",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CanonicalizeURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanonicalizeURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("CanonicalizeURL()\n  got  = %v\n  want = %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeURL_ExpandedTrackingParams(t *testing.T) {
	// Verify that NormalizeURL also strips the newly added tracking params
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Spotify pt param",
			input: "https://open.spotify.com/track/abc?pt=123",
			want:  "https://open.spotify.com/track/abc",
		},
		{
			name:  "Spotify pi param",
			input: "https://open.spotify.com/track/abc?pi=456",
			want:  "https://open.spotify.com/track/abc",
		},
		{
			name:  "YouTube ab_channel param",
			input: "https://youtube.com/watch?v=abc&ab_channel=test",
			want:  "https://youtube.com/watch?v=abc",
		},
		{
			name:  "YouTube feature param",
			input: "https://youtube.com/watch?v=abc&feature=share",
			want:  "https://youtube.com/watch?v=abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeURL(tt.input)
			if err != nil {
				t.Errorf("NormalizeURL() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeURL()\n  got  = %v\n  want = %v", got, tt.want)
			}
		})
	}
}
