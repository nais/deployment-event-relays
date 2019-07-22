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
