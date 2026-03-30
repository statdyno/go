package statdyno

import (
	"errors"
	"log"
)

type Tags map[string]string

type CountStat struct {
	Name  string `json:"stat"`
	Count int    `json:"count"`
	Tags  Tags   `json:"tags,omitempty"`
}

type ValueStat struct {
	Name  string  `json:"stat"`
	Value float64 `json:"value"`
	Tags  Tags    `json:"tags,omitempty"`
}

type MultiStats struct {
	Counts []CountStat `json:"counts,omitempty"`
	Values []ValueStat `json:"values,omitempty"`
}

type Handler interface {
	HandleCount(stat CountStat) error
	HandleValue(stat ValueStat) error
}

type Wrapper struct {
	Handler
}

func (w Wrapper) CountWithTags(name string, count int, tags Tags) error {
	return w.HandleCount(CountStat{Name: name, Count: count, Tags: tags})
}

func (w Wrapper) Count(name string, count int) error {
	return w.HandleCount(CountStat{Name: name, Count: count})
}

func (w Wrapper) ValueWithTags(name string, value float64, tags Tags) error {
	return w.HandleValue(ValueStat{Name: name, Value: value, Tags: tags})
}

func (w Wrapper) Value(name string, value float64) error {
	return w.HandleValue(ValueStat{Name: name, Value: value})
}

var (
	defaultHandler Handler
	defaultWrapper Wrapper
)

func SetDefaultHandler(h Handler) {
	defaultHandler = h
	defaultWrapper = Wrapper{h}
}

func CountWithTags(name string, count int, tags Tags) error {
	return defaultWrapper.CountWithTags(name, count, tags)
}

func Count(name string, count int) error {
	return defaultWrapper.Count(name, count)
}

func ValueWithTags(name string, value float64, tags Tags) error {
	return defaultWrapper.ValueWithTags(name, value, tags)
}

func Value(name string, value float64) error {
	return defaultWrapper.Value(name, value)
}

type NullHandler struct{}

func (h NullHandler) HandleCount(stat CountStat) error {
	return nil
}

func (h NullHandler) HandleValue(stat ValueStat) error {
	return nil
}

var _ Handler = NullHandler{}

type LogHandler struct{}

func (h LogHandler) HandleCount(stat CountStat) error {
	log.Printf("counter stat: name: %s, count: %d, tags: %v", stat.Name, stat.Count, stat.Tags)
	return nil
}

func (h LogHandler) HandleValue(stat ValueStat) error {
	log.Printf("value stat: name: %s, value: %f, tags: %v", stat.Name, stat.Value, stat.Tags)
	return nil
}

var _ Handler = LogHandler{}

type MultiHandler struct {
	multi []Handler
}

func (h *MultiHandler) HandleCount(stat CountStat) error {
	var errs []error
	for i := range h.multi {
		if err := h.multi[i].HandleCount(stat); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (h *MultiHandler) HandleValue(stat ValueStat) error {
	var errs []error
	for i := range h.multi {
		if err := h.multi[i].HandleValue(stat); err != nil {
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
	SetDefaultHandler(NullHandler{})
}
