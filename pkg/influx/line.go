package influx

import (
	"bytes"
	"fmt"
	"github.com/navikt/deployment-event-relays/pkg/deployment"
	"strconv"
	"time"
)

const (
	measurementName = "nais.deployment"
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

// NewLine collects data from a deployment event and copies it to a InfluxDB line data structure.
// Tags will be indexed, but fields will be subject to a table scan if queried.
func NewLine(event *deployment.Event) Line {

	tags := TagField(event.Flatten())

	return Line{
		Measurement: measurementName,
		Timestamp:   event.GetTimestampAsTime(),
		Tags: tags.Selection([]string{
			"application",
			"cluster",
			"environment",
			"namespace",
			"platform_type",
			"rollout_status",
			"team",
		}),
		Fields: tags.Selection([]string{
			"correlation_id",
			"deployer_email",
			"deployer_ident",
			"deployer_name",
			"image_hash",
			"image_name",
			"image_tag",
			"skya_environment",
			"source",
			"version",
		}),
	}
}

// Marshal encodes the data using the InfluxDB line syntax.
// If the data can be serialized, returns a byte slice with a terminating newline.
// If any error occured, the byte slice will be nil, and an error returned.
//
// Field and tag keys will be sorted before serialization.
//
// String fields must be quoted, but tag values are better left unquoted
// as the quotes will appear in the actual data.
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
	writer.WriteString(fmt.Sprintf("%s=%s", keys[0], strconv.Quote(line.Fields[keys[0]])))
	for _, key := range keys[1:] {
		writer.WriteString(fmt.Sprintf(",%s=%s", key, strconv.Quote(line.Fields[key])))
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
