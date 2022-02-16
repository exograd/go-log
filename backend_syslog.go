// Copyright (c) 2022 Exograd SAS.
//
// Permission to use, copy, modify, and/or distribute this software for
// any purpose with or without fee is hereby granted, provided that the
// above copyright notice and this permission notice appear in all
// copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL
// WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE
// AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL
// DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR
// PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER
// TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
// PERFORMANCE OF THIS SOFTWARE.

package log

import (
	"bytes"
	"fmt"
	"os"
	"time"
)

const (
	BOM          = string('\uFEFF')
	FacilityCode = 16 // local use 0
)

type SyslogBackendCfg struct {
	Hostname        string `json:"hostname"`
	Port            int    `json:"port"`
	ApplicationName string `json:"application_name"`
}

type SyslogBackend struct {
	Cfg SyslogBackendCfg
}

func (b *SyslogBackend) Log(msg Message) {
	var buf bytes.Buffer

	// https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.1
	pri := FacilityCode*8 + getSeverityCode(msg.Level)

	// https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.2
	version := 1

	// https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.3
	datetime := msg.Time.Format(time.RFC3339Nano)

	// https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.4
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "-"
	}

	// https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.5
	appname := "-"
	if b.Cfg.ApplicationName != "" {
		appname = b.Cfg.ApplicationName
	}

	// https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.6
	procid := os.Getpid()

	// https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.7
	msgid := "-"

	// https://datatracker.ietf.org/doc/html/rfc5424#section-6.3.1
	sdElementId := "go-log@32473"
	var sdElementParameters bytes.Buffer

	for key, value := range msg.Data {
		sdElementParameters.WriteString(key)
		sdElementParameters.WriteRune('=')
		sdElementParameters.WriteRune('"')
		sdElementParameters.WriteString(
			escapeSdElementValue(formatDatum2(value)))
		sdElementParameters.WriteRune('"')
	}

	// https://datatracker.ietf.org/doc/html/rfc5424#section-6.4
	message := BOM + msg.Message

	var format string
	var arguments []interface{}

	// https://datatracker.ietf.org/doc/html/rfc5424#section-6
	if sdElementParameters.Len() == 0 {
		format = "<%d> %d %s %s %s %s %s [%s] %s"
		arguments = []interface{}{pri, version, datetime,
			hostname, appname, procid, msgid, sdElementId,
			message}
	} else {
		format = "<%d> %d %s %s %s %s %s [%s %s] %s"
		arguments = []interface{}{pri, version, datetime,
			hostname, appname, procid, msgid, sdElementId,
			sdElementParameters.String(), message}
	}

	fmt.Fprintf(&buf, format, arguments...)
}

func getSeverityCode(l Level) int {
	var code int

	switch l {
	case LevelDebug:
		code = 7
	case LevelInfo:
		code = 6
	case LevelError:
		code = 3
	}

	return code
}

func escapeSdElementValue(src string) string {
	var dest bytes.Buffer

	for _, rune := range src {
		switch rune {
		case '\\':
			dest.WriteString("\\\\")
		case '"':
			dest.WriteString("\\\"")
		case ']':
			dest.WriteString("\\]")
		default:
			dest.WriteRune(rune)
		}
	}

	return dest.String()
}

func formatDatum2(datum Datum) string {
	switch v := datum.(type) {
	case fmt.Stringer:
		return formatDatum2(v.String())
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
