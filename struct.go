package flagx

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/tpack"
)

// struct tags are used by *FlagSet.StructVars.
const (
	tagNameFlag       = "flag"
	tagKeyOmit        = "-"
	tagKeyNameDefault = "def"
	tagKeyNameUsage   = "usage"
)

var timeDurationTypeID = tpack.Unpack(time.Duration(0)).RuntimeTypeID()

func (f *FlagSet) varFromStruct(v reflect.Value) error {
	v = goutil.DereferenceValue(v)
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("flagx: want struct pointer field, but got %s", v.Type().String())
	}
	t := v.Type()
	for i := t.NumField() - 1; i >= 0; i-- {
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
		ft := t.Field(i)
		tag, ok := ft.Tag.Lookup(tagNameFlag)
		if !ok || tag == tagKeyOmit {
			continue
		}
		ftElem := goutil.DereferenceType(ft.Type)
		switch ftElem.Kind() {
		case reflect.String,
			reflect.Bool,
			reflect.Float64,
			reflect.Int, reflect.Int64,
			reflect.Uint, reflect.Uint64:
		default:
			return fmt.Errorf("flagx: not support field type %s", ft.Type.String())
		}
		fvElem := goutil.DereferenceValue(fv)
		keys := strings.SplitN(tag, ";", 3)
		var def, usage string
		var names []string
		for _, key := range keys {
			key = strings.TrimSpace(key)
			_def, ok := parseTagKey(key, tagKeyNameDefault)
			if ok {
				def = _def
				continue
			}
			_usage, ok := parseTagKey(key, tagKeyNameUsage)
			if ok {
				usage = _usage
				continue
			}
			names = parseTagNames(key)
		}
		if len(names) == 0 {
			names = append(names, ft.Name)
		}
		err := f.varReflectValue(fvElem, names, def, usage)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *FlagSet) varReflectValue(elem reflect.Value, names []string, def, usage string) error {
	var err error
	val := elem.Addr().Interface()
	kind := elem.Kind()
	switch kind {
	case reflect.String:
		for _, name := range names {
			f.FlagSet.StringVar(val.(*string), name, def, usage)
		}
	case reflect.Bool:
		var b bool
		if def != "" {
			b, err = strconv.ParseBool(def)
			if err != nil {
				return fmt.Errorf("flagx: %q cannot be converted to bool", def)
			}
		}
		for _, name := range names {
			f.FlagSet.BoolVar(val.(*bool), name, b, usage)
		}
	case reflect.Float64:
		var b float64
		if def != "" {
			b, err = strconv.ParseFloat(def, 64)
			if err != nil {
				return fmt.Errorf("flagx: %q cannot be converted to float64", def)
			}
		}
		for _, name := range names {
			f.FlagSet.Float64Var(val.(*float64), name, b, usage)
		}
	case reflect.Int:
		var b int
		if def != "" {
			b, err = strconv.Atoi(def)
			if err != nil {
				return fmt.Errorf("flagx: %q cannot be converted to int", def)
			}
		}
		for _, name := range names {
			f.FlagSet.IntVar(val.(*int), name, b, usage)
		}
	case reflect.Int64:
		if tpack.RuntimeTypeID(elem.Type()) == timeDurationTypeID {
			var b time.Duration
			if def != "" {
				b, err = time.ParseDuration(def)
				if err != nil {
					return fmt.Errorf("flagx: %q cannot be converted to time.Duration", def)
				}
			}
			for _, name := range names {
				f.FlagSet.DurationVar(val.(*time.Duration), name, b, usage)
			}
		} else {
			var b int64
			if def != "" {
				b, err = strconv.ParseInt(def, 10, 64)
				if err != nil {
					return fmt.Errorf("flagx: %q cannot be converted to int64", def)
				}
			}
			for _, name := range names {
				f.FlagSet.Int64Var(val.(*int64), name, b, usage)
			}
		}
	case reflect.Uint:
		var b uint
		if def != "" {
			b2, err := strconv.ParseUint(def, 10, 64)
			if err != nil {
				return fmt.Errorf("flagx: %q cannot be converted to uint", def)
			}
			b = uint(b2)
		}
		for _, name := range names {
			f.FlagSet.UintVar(val.(*uint), name, b, usage)
		}
	case reflect.Uint64:
		var b uint64
		if def != "" {
			b, err = strconv.ParseUint(def, 10, 64)
			if err != nil {
				return fmt.Errorf("flagx: %q cannot be converted to uint64", def)
			}
		}
		for _, name := range names {
			f.FlagSet.Uint64Var(val.(*uint64), name, b, usage)
		}
	default:
		return fmt.Errorf("flagx: not support field type %s", elem.Type().String())
	}
	return nil
}

func parseTagKey(key, keyName string) (string, bool) {
	v := strings.TrimPrefix(key, keyName+"=")
	if v == key {
		v = strings.TrimPrefix(key, keyName+" =")
	}
	if v == key {
		return "", false
	}
	return strings.TrimSpace(v), true
}

func parseTagNames(key string) []string {
	a := strings.Split(key, ",")
	names := make([]string, 0, len(a))
	for _, s := range a {
		s = strings.TrimSpace(s)
		if s != "" {
			names = append(names, s)
		}
	}
	return names
}
