package ethtest

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

func EnableLogging(t *testing.T) {
	log.SetDefault(log.NewLogger(&slogHandler{t: t}))
}

type slogHandler struct {
	t        *testing.T
	attrs    []slog.Attr
	groups   []string
	groupKey string
}

func (s *slogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (s *slogHandler) Handle(_ context.Context, r slog.Record) error {
	var attrs []string
	appendAttr := func(attr slog.Attr) {
		if attr.Equal(slog.Attr{}) {
			return
		}
		attrs = append(attrs, fmt.Sprintf("%s=%q", s.groupKey+attr.Key, attr.Value))
	}

	for _, attr := range s.attrs {
		appendAttr(attr)
	}

	r.Attrs(func(attr slog.Attr) bool {
		appendAttr(attr)
		return true
	})

	if len(attrs) > 0 {
		s.t.Logf("[%s] (%s) %s %s", r.Time, r.Level, r.Message, strings.Join(attrs, ","))
	} else {
		s.t.Logf("[%s] (%s) %s", r.Time, r.Level, r.Message)
	}
	return nil
}

func (s *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	dup := s.dup()
	dup.attrs = append(dup.attrs, attrs...)
	return dup
}

func (s *slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return s
	}
	dup := s.dup()
	dup.groups = append(dup.groups, name)
	dup.groupKey = strings.Join(dup.groups, ".") + "."
	return dup
}

func (s *slogHandler) dup() *slogHandler {
	return &slogHandler{
		t:      s.t,
		attrs:  append([]slog.Attr(nil), s.attrs...),
		groups: append([]string(nil), s.groups...),
	}
}
