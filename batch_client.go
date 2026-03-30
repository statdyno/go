package statdyno

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type runningAverage struct {
	ValueStat
	N int
}

func (ra *runningAverage) Add(value float64) {
	ra.N++
	ra.Value += (value - ra.Value) / float64(ra.N)
}

func encodeTags(tags Tags) string {
	if len(tags) == 0 {
		return ""
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var buf strings.Builder
	for i, key := range keys {
		buf.WriteString(key)
		buf.WriteRune('=')
		buf.WriteString(tags[key])
		if i < len(tags)-1 {
			buf.WriteRune(',')
		}
	}
	return buf.String()
}

func cacheKey(name string, tags Tags) string {
	return name + ":" + encodeTags(tags)
}

type BatchClient struct {
	*Client

	interval   time.Duration
	stop       chan any
	countCache map[string]*CountStat
	valueCache map[string]*runningAverage
	cacheLck   sync.Mutex
}

func (c *BatchClient) HandleCount(stat CountStat) error {
	if c.shuttingDown() {
		return ErrClientClosed
	}
	c.cacheLck.Lock()
	defer c.cacheLck.Unlock()
	key := cacheKey(stat.Name, stat.Tags)
	if count, ok := c.countCache[key]; ok {
		count.Count += stat.Count
	} else {
		c.countCache[key] = &stat
	}
	return nil
}

func (c *BatchClient) HandleValue(stat ValueStat) error {
	if c.shuttingDown() {
		return ErrClientClosed
	}
	c.cacheLck.Lock()
	defer c.cacheLck.Unlock()
	key := cacheKey(stat.Name, stat.Tags)
	if value, ok := c.valueCache[key]; ok {
		value.Add(stat.Value)
	} else {
		c.valueCache[key] = &runningAverage{stat, 1}
	}
	return nil
}

func (c *BatchClient) postStats() {
	c.cacheLck.Lock()
	defer c.cacheLck.Unlock()
	if len(c.countCache) == 0 && len(c.valueCache) == 0 {
		return // Nothing to do
	}
	var stats MultiStats
	if n := len(c.countCache); n > 0 {
		stats.Counts = make([]CountStat, 0, n)
		for _, count := range c.countCache {
			stats.Counts = append(stats.Counts, *count)
		}
		clear(c.countCache)
	}
	if n := len(c.valueCache); n > 0 {
		stats.Values = make([]ValueStat, 0, n)
		for _, value := range c.valueCache {
			stats.Values = append(stats.Values, value.ValueStat)
		}
		clear(c.valueCache)
	}
	c.wg.Go(func() { c.post(stats) })
}

func (c *BatchClient) Shutdown(ctx context.Context) error {
	c.stop <- true
	c.postStats()
	return c.Client.Shutdown(ctx)
}

func NewBatchClient(authToken string, interval time.Duration) *BatchClient {
	client := New(authToken)
	bc := &BatchClient{
		Client:     client,
		interval:   interval,
		stop:       make(chan any),
		countCache: make(map[string]*CountStat),
		valueCache: make(map[string]*runningAverage),
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				bc.postStats()
			case <-bc.stop:
				return
			}
		}
	}()
	return bc
}
