package influx

import (
	"bytes"
	"fmt"
	"strconv"
	"time"
)

// Line represents a single data entry in the InfluxDB line protocol.
//
// https://docs.influxdata.com/influxdb/v1.7/write_protocols/line_protocol_reference/
//
type Line struct {
	Measurement string
	Tags        TagField
	Fields      TagField
	Timestamp   time.Time
}

// Marshal encodes the data using the InfluxDB line syntax.
// If the data can be serialized, returns a byte slice with a terminating newline.
// If any error occured, the byte slice will be nil, and an error returned.
//
// Field and tag keys will be sorted before serialization.
//
// The InfluxDB line format is as follows:
//
// <measurement>[,<tag_key>=<tag_value>[,<tag_key>=<tag_value>]] <field_key>=<field_value>[,<field_key>=<field_value>] [<timestamp>]
//
func (line Line) Marshal() ([]byte, error) {
	if len(line.Measurement) == 0 {
		return nil, fmt.Errorf("InfluxDB line format requires a measurement name")
	}

	if len(line.Fields) == 0 {
		return nil, fmt.Errorf("InfluxDB line format requires at least one field in a measurement")
	}

	buf := bytes.NewBuffer([]byte{})
	writer := errorWriter{w: buf}

	// <measurement>
	writer.WriteString(line.Measurement)

	// [,<tag_key>=<tag_value>[,<tag_key>=<tag_value>]]
	for _, key := range line.Tags.Sorted() {
		writer.WriteString(fmt.Sprintf(",%s=%s", key, line.Tags[key]))
	}

	// DELIMITER
	writer.WriteString(" ")

	// <field_key>=<field_value>[,<field_key>=<field_value>]
	keys := line.Fields.Sorted()
	writer.WriteString(fmt.Sprintf("%s=%s", keys[0], line.Fields[keys[0]]))
	for _, key := range keys[1:] {
		writer.WriteString(fmt.Sprintf(",%s=%s", key, line.Fields[key]))
	}

	// DELIMITER
	writer.WriteString(" ")

	// [<timestamp>]
	// Unix nanosecond timestamp. Specify alternative precisions with the InfluxDB API.
	t := line.Timestamp.Truncate(time.Second)
	writer.WriteString(strconv.FormatInt(t.UnixNano(), 10))

	writer.WriteString("\n")

	return writer.Result()
}
