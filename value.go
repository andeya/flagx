package flagx

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/henrylee2cn/goutil/tpack"
)

func newAnyValue(elem reflect.Value) (Value, error) {
	kind := elem.Kind()
	switch kind {
	case reflect.String:
		return &stringValue{elem}, nil
	case reflect.Bool:
		return &boolValue{elem}, nil
	case reflect.Float64:
		return &boolValue{elem}, nil
	case reflect.Int:
		return &intValue{elem: elem, typeStr: "int"}, nil
	case reflect.Int64:
		if tpack.RuntimeTypeID(elem.Type()) == timeDurationTypeID {
			return &durationValue{elem}, nil
		}
		return &intValue{elem: elem, typeStr: "int64"}, nil
	case reflect.Uint:
		return &uintValue{elem: elem, typeStr: "uint"}, nil
	case reflect.Uint64:
		return &uintValue{elem: elem, typeStr: "uint64"}, nil
	}
	return nil, fmt.Errorf("flagx: not support field type %s", elem.Type().String())
}

type stringValue struct {
	elem reflect.Value
}

func (v *stringValue) String() string {
	if !v.elem.IsValid() {
		return ""
	}
	return v.elem.String()
}

func (v *stringValue) Set(val string) error {
	v.elem.SetString(val)
	return nil
}

type boolValue struct {
	elem reflect.Value
}

func (v *boolValue) String() string {
	if !v.elem.IsValid() {
		return "false"
	}
	return strconv.FormatBool(v.elem.Bool())
}

func (v *boolValue) Set(val string) error {
	if val == "" {
		v.elem.SetBool(false)
		return nil
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return fmt.Errorf("flagx: %q cannot be converted to bool", val)
	}
	v.elem.SetBool(b)
	return nil
}

type float64Value struct {
	elem reflect.Value
}

func (v *float64Value) String() string {
	if !v.elem.IsValid() {
		return "0"
	}
	return strconv.FormatFloat(v.elem.Float(), 'g', -1, 64)
}

func (v *float64Value) Set(val string) error {
	if val == "" {
		v.elem.SetFloat(0)
		return nil
	}
	b, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fmt.Errorf("flagx: %q cannot be converted to float64", val)
	}
	v.elem.SetFloat(b)
	return nil
}

type intValue struct {
	elem    reflect.Value
	typeStr string
}

func (v *intValue) String() string {
	if !v.elem.IsValid() {
		return "0"
	}
	return strconv.FormatInt(v.elem.Int(), 10)
}

func (v *intValue) Set(val string) error {
	if val == "" {
		v.elem.SetInt(0)
		return nil
	}
	b, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return fmt.Errorf("flagx: %q cannot be converted to %s", val, v.typeStr)
	}
	v.elem.SetInt(int64(b))
	return nil
}

type durationValue struct {
	elem reflect.Value
}

func (v *durationValue) String() string {
	if !v.elem.IsValid() {
		return "0s"
	}
	return (time.Duration)(v.elem.Int()).String()
}

func (v *durationValue) Set(val string) error {
	if val == "" {
		v.elem.SetInt(0)
		return nil
	}
	b, err := time.ParseDuration(val)
	if err != nil {
		return fmt.Errorf("flagx: %q cannot be converted to time.Duration", val)
	}
	v.elem.SetInt(int64(b))
	return nil
}

type uintValue struct {
	elem    reflect.Value
	typeStr string
}

func (v *uintValue) String() string {
	if !v.elem.IsValid() {
		return "0"
	}
	return strconv.FormatUint(v.elem.Uint(), 10)
}

func (v *uintValue) Set(val string) error {
	if val == "" {
		v.elem.SetUint(0)
		return nil
	}
	b, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return fmt.Errorf("flagx: %q cannot be converted to %s", val, v.typeStr)
	}
	v.elem.SetUint(b)
	return nil
}