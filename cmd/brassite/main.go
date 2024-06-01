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
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/mmcdole/gofeed"
	slogmulti "github.com/samber/slog-multi"
	"github.com/teknologi-umum/brassite"
)

var version string
var environment = os.Getenv("ENVIRONMENT")

func main() {
	// This is a very simple program, you can extend this to any extend you'd like.
	// 1. Read configuration file
	// 2. For each feed, create a goroutine that will check the feed every `Interval` duration
	// 3. If there's a new item, send it to the delivery routes
	// 4. If there's an error, log it
	var configFilePath string
	flag.StringVar(&configFilePath, "config", "", "Path to the configuration file")
	var sentryDsn string
	flag.StringVar(&sentryDsn, "sentry-dsn", "", "Sentry DSN")
	var logLevel string
	flag.StringVar(&logLevel, "log-level", "warn", "Log level")
	var logPretty bool
	flag.BoolVar(&logPretty, "log-pretty", false, "Log pretty")
	flag.Parse()

	var slogLevel slog.Level
	switch logLevel {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	}
	if logPretty {
		slog.SetDefault(slog.New(slogmulti.Fanout(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slogLevel}),
			NewSlogSentryBreadcrumbsHandler(),
		)))
	} else {
		slog.SetDefault(slog.New(slogmulti.Fanout(
			slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slogLevel}),
			NewSlogSentryBreadcrumbsHandler(),
		)))
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              sentryDsn,
		SampleRate:       1.0,
		EnableTracing:    true,
		TracesSampleRate: 0.2,
		Environment:      environment,
		Release:          version,
	}); err != nil {
		slog.Error("Failed to initialize Sentry", slog.Any("error", err))
		os.Exit(70)
		return
	}

	slog.Debug("Reading from configuration file", slog.String("path", configFilePath))

	config, err := brassite.ParseConfiguration(configFilePath)
	if err != nil {
		slog.Error("Failed to parse configuration", slog.Any("error", err))
		os.Exit(69) // hehe
		return
	}

	// Validate the configuration
	ok, errs := config.Validate()
	if !ok {
		slog.Error("Configuration is invalid", slog.Any("errors", errs))
		os.Exit(68)
		return
	}

	slog.Debug("Configuration is valid")
	slog.Info("Starting Brassite")

	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, os.Interrupt, syscall.SIGTERM)

	for _, feed := range config.Feeds {
		go runWorker(feed)
	}

	<-exitSignal
	slog.Info("Shutting down Brassite")
}

func runWorker(feed brassite.Feed) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetTag("feed_name", feed.Name)
		hub.Scope().SetExtras(map[string]interface{}{
			"feed_name":       feed.Name,
			"url":             feed.URL,
			"interval":        feed.Interval.String(),
			"without_content": feed.WithoutContent,
		})
		ctx = sentry.SetHubOnContext(ctx, hub)

		slog.DebugContext(ctx, "Starting worker", slog.String("feed_name", feed.Name), slog.String("url", feed.URL), slog.Duration("interval", feed.Interval))

		// Call the feed parser
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, feed.URL, nil)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to create request", slog.Any("error", err), slog.String("feed_name", feed.Name))
			cancel()
			sentry.GetHubFromContext(ctx).CaptureException(err)
			time.Sleep(feed.Interval)
			continue
		}

		request.Header.Add("Accept", "*/*")
		request.Header.Add("User-Agent", "Brassite/1.0")

		for key, value := range feed.Headers {
			request.Header.Add(key, value)
		}

		if feed.BasicAuth.Username != "" || feed.BasicAuth.Password != "" {
			request.SetBasicAuth(feed.BasicAuth.Username, feed.BasicAuth.Password)
		}

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to send request", slog.Any("error", err), slog.String("feed_name", feed.Name))
			cancel()
			sentry.GetHubFromContext(ctx).CaptureException(err)
			time.Sleep(feed.Interval)
			continue
		}

		slog.DebugContext(ctx, "Received response", slog.String("feed_name", feed.Name), slog.Int("status_code", response.StatusCode), slog.String("content_type", response.Header.Get("Content-Type")))

		parser := gofeed.NewParser()
		remoteFeed, err := parser.Parse(response.Body)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to parse feed", slog.Any("error", err), slog.String("feed_name", feed.Name))
			_ = response.Body.Close()
			cancel()
			sentry.GetHubFromContext(ctx).CaptureException(err)
			time.Sleep(feed.Interval)
			continue
		}

		// Don't take too long to close the body
		_ = response.Body.Close()

		// Only select the new items by using now - interval
		var newItems []*gofeed.Item
		for _, item := range remoteFeed.Items {
			slog.DebugContext(ctx, "Parsing item", slog.String("feed_name", feed.Name), slog.String("item_title", item.Title), slog.String("item_link", item.Link))

			if item.PublishedParsed != nil {
				slog.DebugContext(ctx, "Published parsed value", slog.String("feed_name", feed.Name), slog.Time("published_parsed", *item.PublishedParsed), slog.Time("now", time.Now().UTC()))
				if item.PublishedParsed.After(time.Now().UTC().Add(-feed.Interval)) {
					newItems = append(newItems, item)
					continue
				}
			}

			if item.UpdatedParsed != nil {
				slog.DebugContext(ctx, "Updated parsed value", slog.String("feed_name", feed.Name), slog.Time("updated_parsed", *item.UpdatedParsed), slog.Time("now", time.Now().UTC()))
				if item.UpdatedParsed.After(time.Now().UTC().Add(-feed.Interval)) {
					newItems = append(newItems, item)
					continue
				}
			}
		}

		slog.DebugContext(ctx, "Found new items", slog.String("feed_name", feed.Name), slog.Int("new_items", len(newItems)))

		// Deliver it
		for _, item := range newItems {
			var itemDate time.Time
			if item.PublishedParsed != nil {
				itemDate = *item.PublishedParsed
			}

			feedItem := brassite.FeedItem{
				ChannelTitle:       remoteFeed.Title,
				ChannelDescription: remoteFeed.Description,
				ChannelURL:         remoteFeed.Link,
				ItemTitle:          item.Title,
				ItemDescription:    item.Description,
				ItemDate:           itemDate.Format(time.Stamp),
				ItemURL:            item.Link,
			}

			if feed.WithoutContent {
				feedItem.ItemDescription = ""
			}

			// Deliver to Discord
			if len(feed.Delivery.DiscordWebhookUrl.Values) > 0 {
				for _, url := range feed.Delivery.DiscordWebhookUrl.Values {
					err := brassite.DeliverToDiscord(ctx, url, feedItem, feed.Logo)
					if err != nil {
						slog.ErrorContext(ctx, "Failed to deliver to Discord", slog.String("feed_name", feed.Name), slog.Any("error", err))

						sentry.GetHubFromContext(ctx).CaptureException(err)
					}
				}
			}

			// TODO: Feel free to submit a PR and work on this
			// Deliver to Telegram
			// if feed.Delivery.TelegramBotToken != "" && feed.Delivery.TelegramChatId != "" {
			// 	err := brassite.DeliverToTelegram(ctx, feed.Delivery.TelegramBotToken, feed.Delivery.TelegramChatId, feedItem)
			// 	if err != nil {
			//      slog.Error("Failed to deliver to Telegram", slog.String("feed_name", feed.Name), slog.Any("error", err))
			// 		sentry.CaptureException(err)
			// 	}
			// }
		}

		cancel()

		time.Sleep(feed.Interval)
	}
}
