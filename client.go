package statdyno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
)

const DefaultServerEndpoint = "https://api.statdyno.com"

var ErrClientClosed = errors.New("client closed")

type ServerError struct {
	StatusCode int
}

func (se ServerError) Error() string {
	return fmt.Sprintf("Expected %d response from server, got %d instead", http.StatusAccepted, se.StatusCode)
}

type Client struct {
	ServerEndpoint   string
	PostErrorHandler func(error)

	authToken  string
	wg         sync.WaitGroup
	inShutdown atomic.Bool
}

func (c *Client) post(stats MultiStats) {
	var err error
	defer func() {
		if err != nil && c.PostErrorHandler != nil {
			c.PostErrorHandler(err)
		}
	}()
	buf := new(bytes.Buffer)
	if err = json.NewEncoder(buf).Encode(stats); err != nil {
		return
	}
	req, err := http.NewRequest("POST", c.ServerEndpoint, buf)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		err = ServerError{resp.StatusCode}
	}
}

func (c *Client) HandleCount(stat CountStat) error {
	if c.shuttingDown() {
		return ErrClientClosed
	}
	c.wg.Go(func() {
		c.post(MultiStats{Counts: []CountStat{stat}})
	})
	return nil
}

func (c *Client) HandleValue(stat ValueStat) error {
	if c.shuttingDown() {
		return ErrClientClosed
	}
	c.wg.Go(func() {
		c.post(MultiStats{Values: []ValueStat{stat}})
	})
	return nil
}

func (c *Client) shuttingDown() bool {
	return c.inShutdown.Load()
}

func (c *Client) Shutdown(ctx context.Context) error {
	c.inShutdown.Store(true)
	wait := func() chan any {
		ch := make(chan any)
		go func() {
			c.wg.Wait()
			ch <- true
		}()
		return ch
	}
	select {
	case <-wait():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

var _ Handler = &Client{}

func New(authToken string) *Client {
	return &Client{
		ServerEndpoint: DefaultServerEndpoint,
		authToken:      authToken,
	}
}
