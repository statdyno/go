package statdyno

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type dummyHandler struct {
	stats MultiStats
}

func (dh *dummyHandler) HandleCount(name string, count int) error {
	dh.stats.Counts = append(dh.stats.Counts, CountStat{Name: name, Count: count})
	return nil
}

func (dh *dummyHandler) HandleValue(name string, value float64) error {
	dh.stats.Values = append(dh.stats.Values, ValueStat{Name: name, Value: value})
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
