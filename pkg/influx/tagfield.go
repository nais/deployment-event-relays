package influx

import (
	"sort"
)

// TagField contains the data for a TAG or a FIELD in InfluxDB.
type TagField map[string]string

// Sorted returns a sorted list of keys in the dictionary.
func (t TagField) Sorted() []string {
	keys := make(sort.StringSlice, len(t))
	index := 0
	for key := range t {
		keys[index] = key
		index++
	}
	keys.Sort()
	return keys
}

// Selection returns a subset of the fields.
// A new TagField with only the provided keys will be returned.
func (t TagField) Selection(keys []string) TagField {
	selection := make(TagField)
	for _, key := range keys {
		if val, ok := t[key]; ok {
			selection[key] = val
		}
	}
	return selection
}
