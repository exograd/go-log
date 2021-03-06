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
	"bytes"
	"encoding/json"
	"fmt"
	stdlog "log"
	"strings"
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

	backendCfg := func(cfgObj interface{}) (interface{}, error) {
		switch {
		case cfg.Backend != nil:
			return cfg.Backend, nil

		case cfg.BackendData != nil:
			if err := json.Unmarshal(*cfg.BackendData, cfgObj); err != nil {
				return nil,
					fmt.Errorf("invalid backend configuration: %w", err)
			}

			return cfgObj, nil
		}

		return cfgObj, nil
	}

	switch cfg.BackendType {
	case BackendTypeTerminal:
		bcfg, err := backendCfg(&TerminalBackendCfg{})
		if err != nil {
			return nil, err
		}
		bcfg2 := bcfg.(*TerminalBackendCfg)
		l.Backend = NewTerminalBackend(*bcfg2)

	case BackendTypeSyslog:
		bcfg, err := backendCfg(&SyslogBackendCfg{})
		if err != nil {
			return nil, err
		}
		bcfg2 := bcfg.(*SyslogBackendCfg)
		l.Backend, err = NewSyslogBackend(*bcfg2)
		if err != nil {
			return nil, fmt.Errorf("cannot create syslog backend: %w", err)
		}

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

func (l *Logger) StdLogger(level Level) *stdlog.Logger {
	// The standard log package does not support log levels, so we have to
	// choose one to be used for all messages.
	//
	// Standard loggers use the io.Writer interface as sink, which does not
	// allow any parameter. We pass the level at the beginning of the message
	// followed by an ASCII unit separator.
	return stdlog.New(l, string(level)+"\x1f", 0)
}

func (l *Logger) Write(data []byte) (int, error) {
	level := LevelInfo
	var msg string

	idx := bytes.IndexByte(data, 0x1f)
	if idx >= 0 {
		isKnownLevel := true

		levelString := string(data[:idx])
		switch levelString {
		case "debug":
			level = LevelDebug
		case "info":
			level = LevelInfo
		case "error":
			level = LevelError
		default:
			isKnownLevel = false
		}

		if isKnownLevel {
			msg = string(data[idx+1:])
		} else {
			msg = string(data)
		}
	}

	msg = strings.TrimSpace(msg)

	l.Log(Message{
		Level:   level,
		Message: msg,
	})

	return len(data), nil
}
