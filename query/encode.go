package query

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const maxLevel = 7

var timeType = reflect.TypeOf(time.Time{})
var timeLayout = "2006-01-02 15:04:05"
var encoderType = reflect.TypeOf(new(Encoder)).Elem()

// SetTimeFormat 设置时间参数的输出格式
func SetTimeFormat(layout string) {
	timeLayout = layout
}

// ScopeOptions 域选项
type ScopeOptions struct {
	Scope string
	Level int
}

type zeroable interface {
	IsZero() bool
}

// Encoder 自定义编码过程
type Encoder interface {
	EncodeValues(scope string, v *url.Values) error
}

// Values 对v进行编码，返回url.Values
func Values(v interface{}) (url.Values, error) {
	values := make(url.Values)

	if v == nil {
		return values, nil
	}

	val := reflect.ValueOf(v)

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return values, nil
		}
		val = val.Elem()
	}

	scope := ScopeOptions{
		Scope: "",
		Level: 1,
	}

	if val.Kind() != reflect.Struct && val.Kind() != reflect.Array && val.Kind() != reflect.Slice && val.Kind() != reflect.Map {
		return nil, fmt.Errorf("unexpects kind: %v", val.Kind())
	}

	err := valueEncode(scope, values, val)
	if err != nil {
		return nil, err
	}

	return values, nil
}

// valueEncode 	解析值
func valueEncode(scope ScopeOptions, values url.Values, val reflect.Value) error {
	if scope.Level > maxLevel {
		return fmt.Errorf("recurse level too deep, the max is: %v", maxLevel)
	}
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// 时间格式
	if val.Type() == timeType {
		t := val.Interface().(time.Time)
		values.Add(scope.Scope, t.Format(timeLayout))
		return nil
	}

	// 自定义Encode方法
	if val.Type().Implements(encoderType) {
		if !reflect.Indirect(val).IsValid() && val.Type().Elem().Implements(encoderType) {
			val = reflect.New(val.Type().Elem())
		}
		m := val.Interface().(Encoder)
		if err := m.EncodeValues(scope.Scope, &values); err != nil {
			return err
		}
		return nil
	}
	var err error
	switch val.Kind() {
	case reflect.Ptr:
		err = valueEncode(scope, values, reflect.ValueOf(val.Interface()))
	case reflect.Struct:
		err = structEncode(scope, values, val)
	case reflect.Slice, reflect.Array:
		err = sliceEncode(scope, values, val)
	case reflect.Map:
		err = mapEncode(scope, values, val)
	case reflect.Interface:
		err = valueEncode(scope, values, reflect.ValueOf(val.Interface()))
	default:
		// 值全部使用fmt输出
		values.Add(scope.Scope, fmt.Sprint(val.Interface()))
	}
	if err != nil {
		return err
	}
	return nil
}

// mapEncode 解析map结构
func mapEncode(scope ScopeOptions, values url.Values, val reflect.Value) error {
	if val.Len() == 0 {
		return nil
	}
	mapInte := val.MapRange()
	for mapInte.Next() {
		k := mapInte.Key()
		if k.Kind() != reflect.String {
			return fmt.Errorf("kind of map key must be string, get: %v", k.Kind())
		}
		key := k.String()
		v := mapInte.Value()
		newScope := scope.Scope + key
		if scope.Scope != "" {
			newScope = scope.Scope + "[" + key + "]"
		}
		err := valueEncode(ScopeOptions{
			Scope: newScope,
			Level: scope.Level + 1,
		}, values, v)
		if err != nil {
			return err
		}
	}
	return nil
}

// structEncode 解析结构体
func structEncode(scope ScopeOptions, values url.Values, val reflect.Value) error {
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		// 私有字段
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}
		sv := val.Field(i)
		tag := sf.Tag.Get("qs")
		// 忽略掉该字段
		if tag == "-" {
			continue
		}
		// 嵌套结构体
		if sf.Anonymous {
			v := reflect.Indirect(sv)
			if v.IsValid() && v.Kind() == reflect.Struct {
				err := valueEncode(scope, values, v)
				if err != nil {
					return err
				}
				continue
			}
		}
		name, opts := parseTag(tag)
		// 忽略零值对象
		if opts.Contains("omitempty") && isEmptyValue(sv) {
			continue
		}
		if name == "" {
			name = sf.Name
		}
		newScope := scope.Scope + name
		if scope.Scope != "" {
			newScope = scope.Scope + "[" + name + "]"
		}
		// 解析值
		err := valueEncode(ScopeOptions{
			Scope: newScope,
			Level: scope.Level + 1,
		}, values, sv)
		if err != nil {
			return err
		}
	}
	return nil
}

// 解析数组、切片的值
func sliceEncode(scope ScopeOptions, values url.Values, val reflect.Value) error {
	// 跳过空slice
	if val.Len() == 0 {
		return nil
	}
	for i := 0; i < val.Len(); i++ {
		err := valueEncode(ScopeOptions{
			Scope: scope.Scope + "[" + strconv.Itoa(i) + "]",
			Level: scope.Level + 1,
		}, values, val.Index(i))
		if err != nil {
			return err
		}
	}
	return nil
}

// tagOptions is the string following a comma in a struct field's "url" tag, or
// the empty string. It does not include the leading comma.
type tagOptions []string

// parseTag splits a struct field's url tag into its name and comma-separated
// options.
func parseTag(tag string) (string, tagOptions) {
	s := strings.Split(tag, ",")
	return s[0], s[1:]
}

// Contains checks whether the tagOptions contains the specified option.
func (o tagOptions) Contains(option string) bool {
	for _, s := range o {
		if s == option {
			return true
		}
	}
	return false
}

// isEmptyValue checks if a value should be considered empty for the purposes
// of omitting fields with the "omitempty" option.
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	if z, ok := v.Interface().(zeroable); ok {
		return z.IsZero()
	}
	return false
}
