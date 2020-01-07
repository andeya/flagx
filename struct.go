package flagx

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/tpack"
)

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

func parseTagNames(key, def string) []string {
	a := strings.Split(key, ",")
	names := make([]string, 0, len(a))
	for _, s := range a {
		names = append(names, strings.TrimSpace(s))
	}
	if names[0] == "" {
		names[0] = def
	}
	return names
}

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
			names = parseTagNames(key, ft.Name)
		}
		for _, name := range names {
			err := f.varReflectValue(fvElem, name, def, usage)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *FlagSet) varReflectValue(elem reflect.Value, name, def, usage string) error {
	val, err := newAnyValue(elem)
	if err != nil {
		return err
	}
	err = val.Set(def)
	if err != nil {
		return errors.New("flagx: def=" + strings.TrimPrefix(err.Error(), "flagx: "))
	}
	f.FlagSet.Var(val, name, usage)
	return nil
}
