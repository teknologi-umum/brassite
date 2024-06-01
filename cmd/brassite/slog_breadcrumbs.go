// Copyright 2024 Teknologi Umum <opensource@teknologiumum.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

type slogSentryBreadcrumbs struct {
	attr  []slog.Attr
	group string
}

// Enabled implements slog.Handler.
func (s *slogSentryBreadcrumbs) Enabled(context.Context, slog.Level) bool {
	return true
}

// Handle implements slog.Handler.
func (s *slogSentryBreadcrumbs) Handle(ctx context.Context, r slog.Record) error {
	if ctx == nil {
		return nil
	}

	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		return nil
	}

	var timestamp = r.Time
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	var data = make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		data[a.Key] = a.Value
		return true
	})

	var group = s.group
	if group == "" {
		group = "console"
	}

	var additionalData = make(sentry.BreadcrumbHint)
	for _, a := range s.attr {
		additionalData[a.Key] = a.Value
	}

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Type:      "default",
		Category:  group,
		Message:   r.Message,
		Data:      data,
		Level:     sentry.Level(strings.ToLower(r.Level.String())),
		Timestamp: timestamp,
	}, &additionalData)

	return nil
}

// WithAttrs implements slog.Handler.
func (s *slogSentryBreadcrumbs) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &slogSentryBreadcrumbs{attr: append(s.attr, attrs...), group: s.group}
}

// WithGroup implements slog.Handler.
func (s *slogSentryBreadcrumbs) WithGroup(name string) slog.Handler {
	return &slogSentryBreadcrumbs{group: name, attr: s.attr}
}

// NewSlogSentryBreadcrumbsHandler creates a new slog handler that sends breadcrumbs to Sentry.
func NewSlogSentryBreadcrumbsHandler() slog.Handler {
	return &slogSentryBreadcrumbs{}
}
