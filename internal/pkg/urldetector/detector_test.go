package urldetector

import (
	"testing"
)

func TestFixMalformedQueryString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Double-encoded Discord URL",
			input: "https://youtube.com/watch?v=D1avYj7q42A?si=5c2KrgyqSfo_0jSE",
			want:  "https://youtube.com/watch?v=D1avYj7q42A&si=5c2KrgyqSfo_0jSE",
		},
		{
			name:  "Already correct URL",
			input: "https://youtube.com/watch?v=D1avYj7q42A&si=5c2KrgyqSfo_0jSE",
			want:  "https://youtube.com/watch?v=D1avYj7q42A&si=5c2KrgyqSfo_0jSE",
		},
		{
			name:  "No query string",
			input: "https://youtube.com/watch",
			want:  "https://youtube.com/watch",
		},
		{
			name:  "Multiple malformed ? in query",
			input: "https://example.com/path?a=1?b=2?c=3",
			want:  "https://example.com/path?a=1&b=2&c=3",
		},
		{
			name:  "URL with fragment",
			input: "https://example.com/path?v=123?si=abc#fragment",
			want:  "https://example.com/path?v=123&si=abc#fragment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fixMalformedQueryString(tt.input)
			if got != tt.want {
				t.Errorf("fixMalformedQueryString() = %v, want %v", got, tt.want)
			}
		})
	}
}
