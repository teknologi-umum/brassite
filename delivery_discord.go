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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"

	md "github.com/JohannesKaufmann/html-to-markdown"
)

type discordWebhookObject struct {
	Username  string               `json:"username"`
	AvatarURL string               `json:"avatar_url"`
	Content   string               `json:"content"`
	Embeds    []discordEmbedObject `json:"embeds,omitempty"`
}

type discordEmbedObject struct {
	Author      discordAuthorObject  `json:"author"`
	Title       string               `json:"title"`
	Url         string               `json:"url"`
	Description string               `json:"description"`
	Color       int                  `json:"color"`
	Fields      []discordFieldObject `json:"fields"`
	Thumbnail   discordUrlObject     `json:"thumbnail"`
	Image       discordUrlObject     `json:"image"`
}

type discordAuthorObject struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	IconURL string `json:"icon_url"`
}

type discordFieldObject struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordUrlObject struct {
	Url string `json:"url"`
}

var discordTemplate = template.Must(template.New("discord").Parse("📰 **{{.Title}}**\n\n{{.Content}}Read more: {{.URL}}"))

type discordTemplateData struct {
	Title   string
	Content string
	URL     string
}

func DeliverToDiscord(ctx context.Context, webhookURL string, feedItem FeedItem, customLogo string) error {
	// Prepare the webhook object
	converter := md.NewConverter("", true, nil)

	content, err := converter.ConvertString(feedItem.ItemDescription)
	if err != nil {
		return fmt.Errorf("failed to convert HTML to markdown: %w", err)
	}

	if len(content) > 0 {
		content += "\n\n"
	}

	var sb strings.Builder
	err = discordTemplate.Execute(&sb, discordTemplateData{
		Title:   feedItem.ItemTitle,
		Content: content,
		URL:     feedItem.ItemURL,
	})
	if err != nil {
		return fmt.Errorf("failed to execute discord template: %w", err)
	}

	webhookObject := discordWebhookObject{
		Username:  feedItem.ChannelTitle,
		AvatarURL: customLogo,
		Content:   sb.String(),
	}

	body, err := json.Marshal(webhookObject)
	if err != nil {
		return fmt.Errorf("failed to marshal discord webhook object: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create discord webhook request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send discord webhook: %w", err)
	}
	defer func() {
		if response.Body != nil {
			_ = response.Body.Close()
		}
	}()

	if response.StatusCode >= 400 {
		responseBody, _ := io.ReadAll(response.Body)

		return fmt.Errorf("discord webhook responded with %d (%s)", response.StatusCode, string(responseBody))
	}

	return nil
}
