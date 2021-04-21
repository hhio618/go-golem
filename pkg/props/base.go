package props

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

//TODO: we can use
type Props map[string]interface{}

// InvalidPropertiesError will be raised by `NewFromProperties(properties)` when given invalid `properties`.
type InvalidPropertiesError struct {
	args []string
}

func (i InvalidPropertiesError) Error() string {
	msg := "Invalid properties"
	if i.args != nil && len(i.args) > 0 {
		msg = fmt.Sprintf("%s {%s}", msg, i.args[0])
	}
	return msg
}

func asList(data interface{}) ([]string, error) {
	v := reflect.TypeOf(data)
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		return mapToStringSlice(data, func(in interface{}) string {
			return fmt.Sprintf("%v", in)
		}), nil
	default:
		var _data interface{}
		err := json.Unmarshal([]byte(data.(string)), _data)
		if err != nil {
			return nil, fmt.Errorf("error when decoding '%s'", data)
		}
		switch reflect.ValueOf(_data).Kind() {
		case reflect.Slice:
			return mapToStringSlice(_data, func(in interface{}) string {
				return fmt.Sprintf("%v", in)
			}), nil
		default:
			return []string{fmt.Sprintf("%v", _data)}, nil
		}
	}
}

func mapToStringSlice(slice interface{}, mapFunc func(interface{}) string) []string {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("mapToStringSlice() given a non-slice type")
	}
	ret := make([]string, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = mapFunc(s.Index(i).Interface())
	}

	return ret
}

/*
Initialize the model from a dictionary representation.

When provided with a dictionary of properties, it will find the matching keys
within it and fill the model fields with the values from the dictionary.

It ignores non-matching keys - i.e. doesn't require filtering of the properties'
dictionary before the model is fed with the data. Thus, several models can be
initialized from the same dictionary and all models will only load their own data.
*/

func FromProperties(props Props, model Model) error {
	// Model predefined keys.
	modelKeys := model.Keys()
	fields, values := getFields(model)
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		val := values[i]
		var opt bool
		if tag, ok := field.Tag.Lookup("field"); ok {
			opt = optional(tag)
		}
		key, ok := modelKeys[field.Name]
		if !ok {
			// Skip on unknown keys.
			continue
		}
		prop, ok := props[key]
		if !ok && !opt {
			return fmt.Errorf("missing 1 required positional argument: '%v'", key)
		}
		// Encoding values.
		if ok {
			if !val.CanSet() && !val.CanAddr() {
				return fmt.Errorf("can't set value for field: %v", field.Name)
			}
			switch pValue := prop.(type) {
			case int64:
				val.SetInt(pValue)
			case float64:
				val.SetFloat(pValue)
			case string:
				val.SetString(pValue)
			default:
				val.Set(reflect.ValueOf(pValue))
			}
			// Call validate method if exists.
			if _, ok := field.Type.MethodByName("Validate"); ok {
				rets := reflect.ValueOf(val.Interface()).MethodByName("Validate").Call([]reflect.Value{})
				err, ok := rets[0].Interface().(error)
				if ok && err != nil {
					return err
				}
			}
		}
	}
	// Do custom mapping last.
	return model.CustomMapping(props)
}

func getFields(iface interface{}) ([]reflect.StructField, []reflect.Value) {
	fields := make([]reflect.StructField, 0)
	values := make([]reflect.Value, 0)

	var ift reflect.Type
	var ifv reflect.Value
	tmp := reflect.ValueOf(iface)
	switch tmp.Type().Kind() {
	case reflect.Ptr:
		ift = reflect.Indirect(tmp).Type()
		ifv = tmp.Elem()
	case reflect.Interface:
		return []reflect.StructField{}, []reflect.Value{}
	}

	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		f := ift.Field(i)
		switch v.Kind() {
		case reflect.Struct:
			f, v := getFields(v.Addr().Interface())
			fields = append(fields, f...)
			values = append(values, v...)
		default:
			fields = append(fields, f)
			values = append(values, v)
		}
	}

	return fields, values
}

// optional checks if an struct field contains an optional tag.
func optional(tag string) bool {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "optional" {
			return true
		}
	}
	return false
}

/*
Model is the base struct from which all property models embed.

Provides helper methods to load the property model data from a dictionary and
to get a mapping of all the keys available in the given model.
*/
type Model interface {
	/*
		:return: a mapping between the model's field names and the property keys

		example:
		```python
		>>> import dataclasses
		>>> import typing
		>>> from yapapi.properties.base import Model
		>>> @dataclasses.dataclass
		... class NodeInfo(Model):
		...     name: typing.Optional[str] = \
		...     dataclasses.field(default=None, metadata={"key": "golem.node.id.name"})
		...
		>>> NodeInfo.keys().name
		'golem.node.id.name'
		```
	*/
	Keys() map[string]string
	CustomMapping(props Props) error
}
