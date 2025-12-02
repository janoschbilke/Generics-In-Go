package utils

import (
	"GoParser/model"
	"reflect"
)

func GetColumns() []string {
	var columns []string
	typeOfCounters := reflect.TypeOf(model.GenericCounters{})
	for i := 0; i < typeOfCounters.NumField(); i++ {
		columns = append(columns, typeOfCounters.Field(i).Tag.Get("json"))
	}
	return columns
}
