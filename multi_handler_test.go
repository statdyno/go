package statdyno

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type dummyHandler struct {
	stats MultiStats
}

func (dh *dummyHandler) HandleCount(stat CountStat) error {
	dh.stats.Counts = append(dh.stats.Counts, stat)
	return nil
}

func (dh *dummyHandler) HandleValue(stat ValueStat) error {
	dh.stats.Values = append(dh.stats.Values, stat)
	return nil
}

func (dh *dummyHandler) Reset() {
	dh.stats.Counts = nil
	dh.stats.Values = nil
}

func TestMultiHandler(t *testing.T) {
	th := &dummyHandler{}
	handler := NewMultiHandler(th, th)
	SetDefault(handler)
	for _, test := range []struct {
		name          string
		log           func()
		expectedStats MultiStats
	}{
		{
			name: "count with tags",
			log:  func() { CountTags("counter", 1, Tags{"foo": "bar"}) },
			expectedStats: MultiStats{
				Counts: []CountStat{
					{Name: "counter", Count: 1, Tags: Tags{"foo": "bar"}},
					{Name: "counter", Count: 1, Tags: Tags{"foo": "bar"}},
				},
			},
		},
		{
			name: "count and value",
			log:  func() { Count("counter", 1); Value("value", 123) },
			expectedStats: MultiStats{
				Counts: []CountStat{{Name: "counter", Count: 1}, {Name: "counter", Count: 1}},
				Values: []ValueStat{{Name: "value", Value: 123}, {Name: "value", Value: 123}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			th.Reset()
			test.log()
			if !cmp.Equal(test.expectedStats, th.stats) {
				t.Fatalf("values are not the same %s", cmp.Diff(test.expectedStats, th.stats))
			}
		})
	}
}
