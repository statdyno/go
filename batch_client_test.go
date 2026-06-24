package statdyno

import (
	"context"
	"math"
	"math/rand"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestRunningAverage(t *testing.T) {
	// https://pkg.go.dev/github.com/google/go-cmp/cmp#example-Option-ApproximateFloats
	opt := cmp.Comparer(func(x, y float64) bool {
		delta := math.Abs(x - y)
		mean := math.Abs(x+y) / 2.0
		return delta/mean < 0.0000001
	})
	randValue := func() float64 {
		return rand.Float64() * 100
	}
	for range 10 {
		v := randValue()
		ra := &runningAverage{ValueStat: ValueStat{Value: v}, N: 1}
		values := []float64{v}
		for range 10 {
			v = randValue()
			ra.Add(v)
			values = append(values, v)
		}
		total := float64(0)
		for _, v := range values {
			total += v
		}
		average := total / float64(len(values))
		if !cmp.Equal(average, ra.Value, opt) {
			t.Fatalf("values are not the same: %s", cmp.Diff(average, ra.Value))
		}
	}
}

func TestBatchClient(t *testing.T) {
	th := &testHandler{secret: "secret"}
	ts := httptest.NewServer(th)
	defer ts.Close()
	for _, test := range []struct {
		name             string
		log              func()
		expectedRequests []MultiStats
	}{
		{
			name: "no stats", // Check that client.Shutdown works
		},
		{
			name:             "count",
			log:              func() { Count("counter", 1) },
			expectedRequests: []MultiStats{{Counts: []CountStat{{Name: "counter", Count: 1}}}},
		},
		{
			name: "multiple counts",
			log: func() {
				for range 10 {
					Count("count", 1)
				}
			},
			expectedRequests: []MultiStats{{Counts: []CountStat{{Name: "count", Count: 10}}}},
		},
		{
			name:             "value",
			log:              func() { Value("value", 123.456) },
			expectedRequests: []MultiStats{{Values: []ValueStat{{Name: "value", Value: 123.456}}}},
		},
		{
			name: "multiple values",
			log: func() {
				for x := range 10 {
					Value("value", float64(x))
				}
			},
			expectedRequests: []MultiStats{{Values: []ValueStat{{Name: "value", Value: 4.5}}}},
		},
		{
			name: "count and value",
			log:  func() { Count("count", 123); Value("value", 456.789) },
			expectedRequests: []MultiStats{
				{
					Counts: []CountStat{{Name: "count", Count: 123}},
					Values: []ValueStat{{Name: "value", Value: 456.789}},
				},
			},
		},
	} {
		th.requests = nil
		interval := time.Duration(100 * time.Millisecond)
		client := NewBatchClient("secret", interval)
		client.ServerEndpoint = ts.URL
		SetDefault(client)
		t.Run(test.name, func(t *testing.T) {
			if test.log != nil {
				test.log()
			}
			if err := client.Shutdown(context.Background()); err != nil {
				t.Fatal(err)
			}
			th.Sort()
			if !cmp.Equal(test.expectedRequests, th.requests) {
				t.Fatalf("values are not the same %s", cmp.Diff(test.expectedRequests, th.requests))
			}
		})
	}
}
