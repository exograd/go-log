// Copyright (c) 2022 Exograd SAS.
//
// Permission to use, copy, modify, and/or distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY
// SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR
// IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package log

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type TerminalBackendCfg struct {
	Color bool `json:"color"`
}

type TerminalBackend struct {
	Cfg TerminalBackendCfg
}

func NewTerminalBackend(cfg TerminalBackendCfg) *TerminalBackend {
	b := &TerminalBackend{
		Cfg: cfg,
	}

	return b
}

func (b *TerminalBackend) Log(msg Message) {
	domain := fmt.Sprintf("%-24s", msg.domain)

	level := string(msg.Level)
	if msg.Level == LevelDebug {
		level += "." + strconv.Itoa(msg.DebugLevel)
	}

	fmt.Fprintf(os.Stderr, "%-7s  %s  %s\n",
		level, b.Colorize(ColorGreen, domain), msg.Message)

	if len(msg.Data) > 0 {
		fmt.Fprintf(os.Stderr, "         ")

		keys := make([]string, len(msg.Data))
		i := 0
		for k := range msg.Data {
			keys[i] = k
			i++
		}
		sort.Strings(keys)

		for i, k := range keys {
			if i > 0 {
				fmt.Fprintf(os.Stderr, " ")
			}

			fmt.Fprintf(os.Stderr, "%s=%s",
				b.Colorize(ColorBlue, k), formatDatum(msg.Data[k]))

			i++
		}

		fmt.Fprintf(os.Stderr, "\n")
	}
}

func (b *TerminalBackend) Colorize(color Color, s string) string {
	if !b.Cfg.Color {
		return s
	}

	return Colorize(color, s)
}

func formatDatum(datum Datum) string {
	switch v := datum.(type) {
	case fmt.Stringer:
		return formatDatum(v.String())

	case string:
		if !strings.Contains(v, " ") {
			return v
		}

		return fmt.Sprintf("%q", v)

	default:
		return fmt.Sprintf("%v", v)
	}
}
