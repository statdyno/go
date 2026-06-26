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

func TestTagsEncoding(t *testing.T) {
	for _, test := range []struct {
		name             string
		tags             Tags
		expectedEncoding string
	}{
		{
			name:             "one tag",
			tags:             Tags{"foo": "bar"},
			expectedEncoding: "foo=bar",
		},
		{
			name:             "multiple tags",
			tags:             Tags{"c": "c", "b": "b", "a": "a"},
			expectedEncoding: "a=a,b=b,c=c",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoding := encodeTags(test.tags)
			if encoding != test.expectedEncoding {
				t.Errorf("got encoding '%s', expected '%s'", encoding, test.expectedEncoding)
			}
		})
	}
}

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
			name:             "value with tags",
			log:              func() { ValueTags("value", 456.789, Tags{"foo": "bar"}) },
			expectedRequests: []MultiStats{{Values: []ValueStat{{Name: "value", Value: 456.789, Tags: Tags{"foo": "bar"}}}}},
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
		{
			name: "counts with tags",
			log: func() {
				for range 10 {
					CountTags("counter one", 1, Tags{"foo": "one"})
					CountTags("counter two", 1, Tags{"foo": "two"})
				}
			},
			expectedRequests: []MultiStats{{Counts: []CountStat{
				{Name: "counter one", Count: 10, Tags: Tags{"foo": "one"}},
				{Name: "counter two", Count: 10, Tags: Tags{"foo": "two"}},
			}}},
		},
		{
			name: "counts and values with tags",
			log: func() {
				for x := range 10 {
					CountTags("counter one", 1, Tags{"foo": "one"})
					CountTags("counter two", 1, Tags{"foo": "two"})
					ValueTags("value one", float64(x), Tags{"foo": "one"})
					ValueTags("value two", float64(x*2), Tags{"foo": "two"})
				}
			},
			expectedRequests: []MultiStats{{
				Counts: []CountStat{
					{Name: "counter one", Count: 10, Tags: Tags{"foo": "one"}},
					{Name: "counter two", Count: 10, Tags: Tags{"foo": "two"}},
				},
				Values: []ValueStat{
					{Name: "value one", Value: 4.5, Tags: Tags{"foo": "one"}},
					{Name: "value two", Value: 9, Tags: Tags{"foo": "two"}},
				},
			}},
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
