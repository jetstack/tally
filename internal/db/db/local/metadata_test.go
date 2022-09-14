package local

import (
	"testing"
	"time"
)

func TestMetadataEqual(t *testing.T) {
	testCases := map[string]struct {
		a    Metadata
		b    Metadata
		want bool
	}{
		"equal": {
			a: Metadata{
				Hash:      "foobar",
				Timestamp: time.Now(),
			},
			b: Metadata{
				Hash: "foobar",
				// Equals only compares the Hash so differences in timestamp
				// should be ignored. Timestamp is only there for informational
				// purposes.
				Timestamp: time.Now().Add(24 * time.Hour),
			},
			want: true,
		},
		"not equal": {
			a: Metadata{
				Hash:      "foobar",
				Timestamp: time.Now(),
			},
			b: Metadata{
				Hash:      "barfoo",
				Timestamp: time.Now().Add(24 * time.Hour),
			},
			want: false,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			got := tc.a.Equal(tc.b)
			if got != tc.want {
				t.Fatalf("unexpected Equals result, wanted %t but got %t", tc.want, got)
			}
		})
	}
}
