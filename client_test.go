package statdyno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func (ms MultiStats) sort() {
	sort.Slice(ms.Counts, func(i, j int) bool {
		return ms.Counts[i].Name < ms.Counts[j].Name
	})
	sort.Slice(ms.Values, func(i, j int) bool {
		return ms.Values[i].Name < ms.Values[j].Name
	})
}

func TestNullHandler(t *testing.T) {
	SetDefault(NullHandler{})
	Count("foo", 123)
	Value("foo", 0.123)
}

type testHandler struct {
	secret   string
	requests []MultiStats
}

func (th testHandler) Sort() {
	for _, request := range th.requests {
		request.sort()
	}
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+h.secret {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var request MultiStats
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h.requests = append(h.requests, request)
	w.WriteHeader(http.StatusAccepted)
}

func TestClient(t *testing.T) {
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
			name:             "count with tags",
			log:              func() { CountTags("counter", 99, Tags{"foo": "bar"}) },
			expectedRequests: []MultiStats{{Counts: []CountStat{{Name: "counter", Count: 99, Tags: Tags{"foo": "bar"}}}}},
		},
		{
			name:             "value",
			log:              func() { Value("value", 123.456) },
			expectedRequests: []MultiStats{{Values: []ValueStat{{Name: "value", Value: 123.456}}}},
		},
		{
			name:             "value with tags",
			log:              func() { ValueTags("value", 456.789, Tags{"foo": "bar"}) },
			expectedRequests: []MultiStats{{Values: []ValueStat{{Name: "value", Value: 456.789, Tags: Tags{"foo": "bar"}}}}},
		},
		{
			name: "multiple counts",
			log:  func() { Count("count 1", 1); time.Sleep(10 * time.Millisecond); Count("count 2", 2) },
			expectedRequests: []MultiStats{
				{Counts: []CountStat{{Name: "count 1", Count: 1}}},
				{Counts: []CountStat{{Name: "count 2", Count: 2}}},
			},
		},
		{
			name: "count and value",
			log:  func() { Count("count", 123); time.Sleep(10 * time.Millisecond); Value("value", 456.789) },
			expectedRequests: []MultiStats{
				{Counts: []CountStat{{Name: "count", Count: 123}}},
				{Values: []ValueStat{{Name: "value", Value: 456.789}}},
			},
		},
	} {
		th.requests = nil
		client := New("secret")
		client.ServerEndpoint = ts.URL
		SetDefault(client)
		t.Run(test.name, func(t *testing.T) {
			if test.log != nil {
				test.log()
			}
			if err := client.Shutdown(context.Background()); err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(test.expectedRequests, th.requests) {
				t.Errorf("values are not the same %s", cmp.Diff(test.expectedRequests, th.requests))
			}
		})
	}
}

func TestClientErrorHandler(t *testing.T) {
	th := &testHandler{secret: "secret"}
	ts := httptest.NewServer(th)
	defer ts.Close()
	client := New("wrong secret")
	client.ServerEndpoint = ts.URL
	var postError error
	client.PostErrorHandler = func(err error) {
		postError = err
	}
	SetDefault(client)
	Count("count", 1)
	if err := client.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	badAuthError := ServerError{http.StatusUnauthorized}
	if postError != badAuthError {
		t.Errorf("Expected %v error, got %v instead", badAuthError, postError)
	}
}

func TestClientShutdown(t *testing.T) {
	th := &testHandler{secret: "secret"}
	ts := httptest.NewServer(th)
	defer ts.Close()
	client := New("secret")
	client.ServerEndpoint = ts.URL
	SetDefault(client)
	Count("count", 1)
	if err := client.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := Count("count", 1); err != ErrClientClosed {
		t.Errorf("Expected %v error, got %v instead", ErrClientClosed, err)
	}
}
