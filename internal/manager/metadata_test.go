package manager

import (
	"testing"
)

func TestMetadataEquals(t *testing.T) {
	testCases := map[string]struct {
		a    Metadata
		b    Metadata
		want bool
	}{
		"equal": {
			a: Metadata{
				SHA256: "foobar",
			},
			b: Metadata{
				SHA256: "foobar",
			},
			want: true,
		},
		"not equal": {
			a: Metadata{
				SHA256: "foobar",
			},
			b: Metadata{
				SHA256: "barfoo",
			},
			want: false,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			got := tc.a.Equals(tc.b)
			if got != tc.want {
				t.Fatalf("unexpected Equals result, wanted %t but got %t", tc.want, got)
			}
		})
	}
}
