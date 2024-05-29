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

package brassite

import (
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/titanous/json5"
	"gopkg.in/yaml.v3"
)

type Configuration struct {
	Feeds []Feed `json:"feeds" yaml:"feeds" toml:"feeds"`
}

type Feed struct {
	// Name of the feed
	Name string `json:"name" yaml:"name" toml:"name"`
	// URL of the feed, can be one of RSS, Atom, or JSON feed
	URL string `json:"url" yaml:"url" toml:"url"`
	// Logo that will be displayed (if you're using Discord). Optional, of course.
	// Can be a URL (starts with `http://` or `https://`, or a local file (starts with `file://`).
	// Won't support direct base64 or hex data. Won't support blob-storage as well (S3, GCS, etc.)
	Logo string `json:"logo" yaml:"logo" toml:"logo"`
	// Interval to check the feed
	Interval time.Duration `json:"interval" yaml:"interval" toml:"interval"`
	// BasicAuth for the feed if it requires authentication
	BasicAuth BasicAuth `json:"basic_auth" yaml:"basic_auth" toml:"basic_auth"`
	// Headers for the feed if it requires custom request headers
	Headers map[string]string `json:"headers" yaml:"headers" toml:"headers"`
	// Delivery routes the feed will be sent to
	Delivery Delivery `json:"delivery" yaml:"delivery" toml:"delivery"`
	// WithoutContent won't include the content of the feed item
	WithoutContent bool `json:"without_content" yaml:"without_content" toml:"without_content"`
}

type BasicAuth struct {
	Username string `json:"username" yaml:"username" toml:"username"`
	Password string `json:"password" yaml:"password" toml:"password"`
}

type Delivery struct {
	// Discord webhook URL
	DiscordWebhookUrl DiscordWebhookUrl `json:"discord_webhook_url" yaml:"discord_webhook_url" toml:"discord_webhook_url"`
	// Telegram bot token
	TelegramBotToken string `json:"telegram_bot_token" yaml:"telegram_bot_token" toml:"telegram_bot_token"`
	// Telegram chat ID
	TelegramChatId string `json:"telegram_chat_id" yaml:"telegram_chat_id" toml:"telegram_chat_id"`
}

type DiscordWebhookUrl struct {
	Values []string
}

// References: https://github.com/go-yaml/yaml/issues/100
//
// Custom unmarshaller to support reading a field as string or array of strings
func (d *DiscordWebhookUrl) UnmarshalYAML(unmarshal func(any) error) error {
	var multi []string
	err := unmarshal(&multi)
	if err != nil {
		var single string
		err := unmarshal(&single)
		if err != nil {
			return err
		}
		d.Values = make([]string, 1)
		d.Values[0] = single
	} else {
		d.Values = multi
	}
	return nil
}

func (d *DiscordWebhookUrl) UnmarshalJSON(data []byte) error {
	var multi []string
	err := json5.Unmarshal(data, &multi)
	if err != nil {
		var single string
		err := json5.Unmarshal(data, &single)
		if err != nil {
			return err
		}
		d.Values = make([]string, 1)
		d.Values[0] = single
	} else {
		d.Values = multi
	}
	return nil
}

func (d *DiscordWebhookUrl) UnmarshalTOML(data any) error {
	multi, ok := data.([]any)
	if ok {
		var multiStrs []string
		for _, item := range multi {
			str, _ := item.(string)
			multiStrs = append(multiStrs, str)
		}
		d.Values = multiStrs
		return nil
	} else if single, ok := data.(string); ok {
		d.Values = make([]string, 1)
		d.Values[0] = single
		return nil
	}

	return fmt.Errorf("provided %T, expected string or []string", data)
}

func ParseConfiguration(configPath string) (Configuration, error) {
	if configPath == "" {
		return Configuration{}, fmt.Errorf("config path is empty")
	}

	file, err := os.Open(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Configuration{}, fmt.Errorf("config file not found")
		}
		return Configuration{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Configuration
	switch path.Ext(configPath) {
	case ".json":
		fallthrough
	case ".json5":
		err = json5.NewDecoder(file).Decode(&config)
	case ".yaml":
		fallthrough
	case ".yml":
		err = yaml.NewDecoder(file).Decode(&config)
	case ".toml":
		_, err = toml.NewDecoder(file).Decode(&config)

	default:
		return Configuration{}, fmt.Errorf("unsupported config file format")
	}

	if err != nil {
		return Configuration{}, fmt.Errorf("failed to decode config file: %w", err)
	}

	return config, nil
}

func (c Configuration) Validate() (ok bool, issues *ValidationError) {
	issues = NewValidationError()
	ok = true

	if len(c.Feeds) == 0 {
		issues.AddIssue("feeds", "at least one feed is required (what are you doing with no feed anyway?)")
		ok = false
	}

	for i, feed := range c.Feeds {
		if feed.Name == "" {
			issues.AddIssue(fmt.Sprintf("feeds.%d.name", i), "name is required")
			ok = false
		}
		if feed.URL == "" {
			issues.AddIssue(fmt.Sprintf("feeds.%d.url", i), "url is required")
			ok = false
		}
		if feed.Interval == 0 {
			issues.AddIssue(fmt.Sprintf("feeds.%d.interval", i), "interval is required")
			ok = false
		} else {
			if feed.Interval < 0 {
				issues.AddIssue(fmt.Sprintf("feeds.%d.name", i), "interval must be greater than 0")
				ok = false
			}
		}
		if len(feed.Delivery.DiscordWebhookUrl.Values) == 0 && feed.Delivery.TelegramBotToken == "" {
			issues.AddIssue(fmt.Sprintf("feeds.%d.delivery", i), "at least one delivery method is required (otherwise what's the point?)")
			ok = false
		}

		if feed.Delivery.TelegramBotToken != "" && feed.Delivery.TelegramChatId == "" {
			issues.AddIssue(fmt.Sprintf("feeds.%d.delivery.telegram_chat_id", i), "telegram chat ID is required if telegram bot token is not empty")
			ok = false
		}
	}

	return
}
