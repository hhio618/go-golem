package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Foo struct {
	Bar string `manners:"3"`
}

func main() {
	f := Foo{"I want a tomato"}
	Say(f.Bar)
	// Output:
	// I want a tomato pretty pretty pretty please
}

func Say(v interface{}) {
	rv := reflect.ValueOf(v)
	t := rv.Type()
	for i := 0; i < t.NumField(); i++ {
		if value, ok := t.Field(i).Tag.Lookup("manners"); ok {
			if rv.Field(i).Kind() == reflect.String {
				handleManners(rv.Field(i).String(), value)
			}
		}
	}
}

func handleManners(fieldValue, tagValue string) {
	var prettyTimes int
	if tagValue == "" || tagValue == "-" {
		prettyTimes = 1
	}
	if i, err := strconv.Atoi(tagValue); err == nil {
		prettyTimes = i
	}

	var sb strings.Builder
	sb.WriteString(fieldValue)
	for i := 0; i < prettyTimes; i++ {
		sb.WriteString(" pretty")
	}
	if prettyTimes > 0 {
		sb.WriteString(" please")
	}

	fmt.Println(sb.String())
}
