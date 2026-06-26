package statdyno

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestVarArgs(t *testing.T) {
	for _, test := range []struct {
		name         string
		varTags      []string
		expectedTags Tags
	}{
		{
			"empty",
			[]string{},
			nil,
		},
		{
			"matching",
			[]string{"one", "foo", "two", "bar"},
			Tags{"one": "foo", "two": "bar"},
		},
		{
			"mismatch",
			[]string{"one", "foo", "two"},
			Tags{"one": "foo", "two": ""},
		},
		{
			"key only",
			[]string{"one"},
			Tags{"one": ""},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if tags := varTags(test.varTags...); !cmp.Equal(tags, test.expectedTags) {
				t.Fatalf("values are not the same: %s", cmp.Diff(tags, test.expectedTags))
			}

		})
	}
}
