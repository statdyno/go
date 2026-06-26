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

func varTags(tags ...string) Tags {
	if len(tags) == 0 {
		return nil
	}
	mtags := make(Tags)
	for len(tags) > 0 {
		key := tags[0]
		val := ""
		if len(tags) > 1 {
			val = tags[1]
		}
		mtags[key] = val
		if len(tags) == 1 {
			break
		}
		tags = tags[2:]
	}
	return mtags
}

func (w Wrapper) CountTags(name string, count int, tags Tags) error {
	return w.HandleCount(CountStat{Name: name, Count: count, Tags: tags})
}

func (w Wrapper) Count(name string, count int, tags ...string) error {
	return w.HandleCount(CountStat{Name: name, Count: count, Tags: varTags(tags...)})
}

func (w Wrapper) ValueTags(name string, value float64, tags Tags) error {
	return w.HandleValue(ValueStat{Name: name, Value: value, Tags: tags})
}

func (w Wrapper) Value(name string, value float64, tags ...string) error {
	return w.HandleValue(ValueStat{Name: name, Value: value, Tags: varTags(tags...)})
}

var defaultWrapper Wrapper

func SetDefault(h Handler) {
	defaultWrapper = Wrapper{h}
}

func CountTags(name string, count int, tags Tags) error {
	return defaultWrapper.CountTags(name, count, tags)
}

func Count(name string, count int, tags ...string) error {
	return defaultWrapper.Count(name, count, tags...)
}

func ValueTags(name string, value float64, tags Tags) error {
	return defaultWrapper.ValueTags(name, value, tags)
}

func Value(name string, value float64, tags ...string) error {
	return defaultWrapper.Value(name, value, tags...)
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
	SetDefault(NullHandler{})
}
