package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/tailscale/hujson"
)

func NewHuJSON[T any](t *T) HuJSON[T] {
	marshal, _ := json.Marshal(t)
	return HuJSON[T]{
		v: string(marshal),
		t: t,
	}
}

func ParseHuJson[T any](v string) (*HuJSON[T], error) {
	ast, err := hujson.Parse([]byte(v))
	if err != nil {
		return nil, err
	}

	ast.Format()
	formatted := string(ast.Pack())
	ast.Standardize()

	t := new(T)
	if err := json.Unmarshal(ast.Pack(), t); err != nil {
		return nil, err
	}
	return &HuJSON[T]{v: formatted, t: t}, nil
}

type HuJSON[T any] struct {
	v string
	t *T
}

func (h *HuJSON[T]) Get() *T {
	return h.t
}

func (h *HuJSON[T]) String() string {
	return h.v
}

func (i *HuJSON[T]) Equal(x *HuJSON[T]) bool {
	if i == nil && x == nil {
		return true
	}
	if (i == nil) != (x == nil) {
		return false
	}
	return i.v == x.v
}

func (h HuJSON[T]) Value() (driver.Value, error) {
	if len(h.v) == 0 {
		return nil, nil
	}
	return h.v, nil
}

func (h *HuJSON[T]) Scan(destination interface{}) error {
	var v string
	switch value := destination.(type) {
	case string:
		v = value
	case []byte:
		v = string(value)
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}

	next, err := hujson.Standardize([]byte(v))
	if err != nil {
		return err
	}
	var n = new(T)
	if err := json.Unmarshal(next, n); err != nil {
		return err
	}
	h.v = v
	h.t = n
	return nil
}
