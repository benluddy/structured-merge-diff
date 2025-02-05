/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package value

import (
	"reflect"
	"testing"
	"time"
)

type CustomValue struct {
	data []byte
}

// MarshalJSON has a value receiver on this type.
func (c CustomValue) MarshalJSON() ([]byte, error) {
	return c.data, nil
}

type CustomPointer struct {
	data []byte
}

// MarshalJSON has a pointer receiver on this type.
func (c *CustomPointer) MarshalJSON() ([]byte, error) {
	return c.data, nil
}

// Mimics https://github.com/kubernetes/apimachinery/blob/master/pkg/apis/meta/v1/time.go.
type Time struct {
	time.Time
}

// ToUnstructured implements the value.UnstructuredConverter interface.
func (t Time) ToUnstructured() interface{} {
	if t.IsZero() {
		return nil
	}
	buf := make([]byte, 0, len(time.RFC3339))
	buf = t.UTC().AppendFormat(buf, time.RFC3339)
	return string(buf)
}

func TestToUnstructured(t *testing.T) {
	testcases := []struct {
		Data     string
		Expected interface{}
	}{
		{Data: `null`, Expected: nil},
		{Data: `true`, Expected: true},
		{Data: `false`, Expected: false},
		{Data: `[]`, Expected: []interface{}{}},
		{Data: `[1]`, Expected: []interface{}{int64(1)}},
		{Data: `{}`, Expected: map[string]interface{}{}},
		{Data: `{"a":1}`, Expected: map[string]interface{}{"a": int64(1)}},
		{Data: `0`, Expected: int64(0)},
		{Data: `0.0`, Expected: float64(0)},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.Data, func(t *testing.T) {
			t.Parallel()
			custom := []interface{}{
				CustomValue{data: []byte(tc.Data)},
				&CustomValue{data: []byte(tc.Data)},
				&CustomPointer{data: []byte(tc.Data)},
			}
			for _, custom := range custom {
				rv := reflect.ValueOf(custom)
				result, err := TypeReflectEntryOf(rv.Type()).ToUnstructured(rv)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(result, tc.Expected) {
					t.Errorf("expected %#v but got %#v", tc.Expected, result)
				}
			}
		})
	}
}

func timePtr(t time.Time) *time.Time { return &t }

func TestTimeToUnstructured(t *testing.T) {
	testcases := []struct {
		Name     string
		Time     *time.Time
		Expected interface{}
	}{
		{Name: "nil", Time: nil, Expected: nil},
		{Name: "zero", Time: &time.Time{}, Expected: nil},
		{Name: "1", Time: timePtr(time.Time{}.Add(time.Second)), Expected: "0001-01-01T00:00:01Z"},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			var time *Time
			rv := reflect.ValueOf(time)
			if tc.Time != nil {
				rv = reflect.ValueOf(Time{Time: *tc.Time})
			}
			result, err := TypeReflectEntryOf(rv.Type()).ToUnstructured(rv)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result, tc.Expected) {
				t.Errorf("expected %#v but got %#v", tc.Expected, result)
			}
		})
	}
}

func TestTypeReflectEntryOf(t *testing.T) {
	testString := ""
	tests := map[string]struct {
		arg  interface{}
		want *TypeReflectCacheEntry
	}{
		"StructWithStringField": {
			arg: struct {
				F1 string `json:"f1"`
			}{},
			want: &TypeReflectCacheEntry{
				structFields: map[string]*FieldCacheEntry{
					"f1": {
						JsonName:  "f1",
						fieldPath: [][]int{{0}},
						fieldType: reflect.TypeOf(testString),
						TypeEntry: &TypeReflectCacheEntry{},
					},
				},
				orderedStructFields: []*FieldCacheEntry{
					{
						JsonName:  "f1",
						fieldPath: [][]int{{0}},
						fieldType: reflect.TypeOf(testString),
						TypeEntry: &TypeReflectCacheEntry{},
					},
				},
			},
		},
		"StructWith*StringFieldOmitempty": {
			arg: struct {
				F1 *string `json:"f1,omitempty"`
			}{},
			want: &TypeReflectCacheEntry{
				structFields: map[string]*FieldCacheEntry{
					"f1": {
						JsonName:    "f1",
						isOmitEmpty: true,
						fieldPath:   [][]int{{0}},
						fieldType:   reflect.TypeOf(&testString),
						TypeEntry:   &TypeReflectCacheEntry{},
					},
				},
				orderedStructFields: []*FieldCacheEntry{
					{
						JsonName:    "f1",
						isOmitEmpty: true,
						fieldPath:   [][]int{{0}},
						fieldType:   reflect.TypeOf(&testString),
						TypeEntry:   &TypeReflectCacheEntry{},
					},
				},
			},
		},
		"StructWithInlinedField": {
			arg: struct {
				F1 string `json:",inline"`
			}{},
			want: &TypeReflectCacheEntry{
				structFields:        map[string]*FieldCacheEntry{},
				orderedStructFields: []*FieldCacheEntry{},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := TypeReflectEntryOf(reflect.TypeOf(tt.arg)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TypeReflectEntryOf() = %v, want %v", got, tt.want)
			}
		})
	}
}
