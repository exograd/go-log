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
	"encoding/json"
	"fmt"
	"time"
)

type LoggerCfg struct {
	BackendType BackendType      `json:"backend_type"`
	BackendData *json.RawMessage `json:"backend,omitempty"`
	Backend     interface{}      `json:"-"`
	DebugLevel  int              `json:"debug_level"`
}

type Logger struct {
	Cfg        LoggerCfg
	Backend    Backend
	Domain     string
	Data       Data
	DebugLevel int
}

func DefaultLogger(name string) *Logger {
	backendCfg := TerminalBackendCfg{
		Color: true,
	}

	backend := NewTerminalBackend(backendCfg)

	return &Logger{
		Cfg:     LoggerCfg{},
		Backend: backend,
		Domain:  name,
		Data:    Data{},
	}
}

func NewLogger(name string, cfg LoggerCfg) (*Logger, error) {
	l := &Logger{
		Cfg: cfg,

		Domain:     name,
		Data:       Data{},
		DebugLevel: cfg.DebugLevel,
	}

	decodeBackendCfg := func(dest interface{}) error {
		if cfg.BackendData != nil {
			if err := json.Unmarshal(*cfg.BackendData, dest); err != nil {
				return fmt.Errorf("invalid backend configuration: %w", err)
			}
		}

		return nil
	}

	switch cfg.BackendType {
	case BackendTypeTerminal:
		var backendCfg TerminalBackendCfg
		if err := decodeBackendCfg(&backendCfg); err != nil {
			return nil, err
		}
		l.Backend = NewTerminalBackend(backendCfg)

	case "":
		return nil, fmt.Errorf("missing or empty backend type")

	default:
		return nil, fmt.Errorf("invalid backend type %q", cfg.BackendType)
	}

	return l, nil
}

func (l *Logger) Child(domain string, data Data) *Logger {
	childDomain := l.Domain
	if domain != "" {
		childDomain += "." + domain
	}

	child := &Logger{
		Cfg:     l.Cfg,
		Backend: l.Backend,

		Domain:     childDomain,
		Data:       MergeData(l.Data, data),
		DebugLevel: l.DebugLevel,
	}

	return child
}

func (l *Logger) Log(msg Message) {
	if msg.Level == LevelDebug && l.DebugLevel < msg.DebugLevel {
		return
	}

	var t time.Time
	if msg.Time == nil {
		t = time.Now()
	} else {
		t = *msg.Time
	}

	t = t.UTC()
	msg.Time = &t

	msg.domain = l.Domain

	if msg.Data == nil {
		msg.Data = make(Data)
	}

	msg.Data = MergeData(l.Data, msg.Data)

	l.Backend.Log(msg)
}

func (l *Logger) Debug(level int, format string, args ...interface{}) {
	l.Log(Message{
		Level:      LevelDebug,
		DebugLevel: level,
		Message:    fmt.Sprintf(format, args...),
	})
}

func (l *Logger) DebugData(data Data, level int, format string, args ...interface{}) {
	l.Log(Message{
		Level:      LevelDebug,
		DebugLevel: level,
		Message:    fmt.Sprintf(format, args...),
		Data:       data,
	})
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.Log(Message{
		Level:   LevelInfo,
		Message: fmt.Sprintf(format, args...),
	})
}

func (l *Logger) InfoData(data Data, format string, args ...interface{}) {
	l.Log(Message{
		Level:   LevelInfo,
		Message: fmt.Sprintf(format, args...),
		Data:    data,
	})
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.Log(Message{
		Level:   LevelError,
		Message: fmt.Sprintf(format, args...),
	})
}

func (l *Logger) ErrorData(data Data, format string, args ...interface{}) {
	l.Log(Message{
		Level:   LevelError,
		Message: fmt.Sprintf(format, args...),
		Data:    data,
	})
}
