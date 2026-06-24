package statdyno

import (
	"errors"
	"log"
)

type CountStat struct {
	Name  string `json:"stat"`
	Count int    `json:"count"`
}

type ValueStat struct {
	Name  string  `json:"stat"`
	Value float64 `json:"value"`
}

type MultiStats struct {
	Counts []CountStat `json:"counts,omitempty"`
	Values []ValueStat `json:"values,omitempty"`
}

type Handler interface {
	HandleCount(name string, count int) error
	HandleValue(name string, value float64) error
}

type Wrapper struct {
	Handler
}

func (w Wrapper) Count(name string, count int) error {
	return w.HandleCount(name, count)
}

func (w Wrapper) Value(name string, value float64) error {
	return w.HandleValue(name, value)
}

var defaultWrapper Wrapper

func SetDefault(h Handler) {
	defaultWrapper = Wrapper{h}
}

func Count(name string, count int) error {
	return defaultWrapper.Count(name, count)
}

func Value(name string, value float64) error {
	return defaultWrapper.Value(name, value)
}

type NullHandler struct{}

func (h NullHandler) HandleCount(name string, count int) error {
	return nil
}

func (h NullHandler) HandleValue(name string, value float64) error {
	return nil
}

var _ Handler = NullHandler{}

type LogHandler struct{}

func (h LogHandler) HandleCount(name string, count int) error {
	log.Printf("counter stat: name: %s, count: %d", name, count)
	return nil
}

func (h LogHandler) HandleValue(name string, value float64) error {
	log.Printf("value stat: name: %s, value: %f", name, value)
	return nil
}

var _ Handler = LogHandler{}

type MultiHandler struct {
	multi []Handler
}

func (h *MultiHandler) HandleCount(name string, count int) error {
	var errs []error
	for i := range h.multi {
		if err := h.multi[i].HandleCount(name, count); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (h *MultiHandler) HandleValue(name string, value float64) error {
	var errs []error
	for i := range h.multi {
		if err := h.multi[i].HandleValue(name, value); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

var _ Handler = &MultiHandler{}

func NewMultiHandler(handlers ...Handler) *MultiHandler {
	h := make([]Handler, len(handlers))
	copy(h, handlers)
	return &MultiHandler{multi: h}
}

func init() {
	SetDefault(NullHandler{})
}
