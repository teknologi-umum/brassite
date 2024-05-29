# Brassite

RSS feed reader that forwards the thing to Discord, Telegram, or any of your choice.

## Usage

Create a `config.yml`, or other config file format, accepted ones are `.json`, `.json5`, `.yaml`, `.yml`, and `.toml`.
The configuration file should have something like this:

```yaml
feeds:
  - name: "Hackernews"
    url: "https://news.ycombinator.com/rss"
    logo: "https://cdn.example.com/hacker-news.png"
    interval: 12h
    without_content: true
    delivery:
      discord_webhook_url: "https://discord.com/api/webhooks/..../...."

  - name: Tech Crunch
    url: "https://techcrunch.com/feed/"
    interval: 5m
    without_content: false
    delivery:
      discord_webhook_url: "https://discord.com/api/webhooks/..../...."
```

Then run the Docker container.

```sh
docker run -v ./config.yml:/config.yml ghcr.io/teknologi-umum/brassite:edge /usr/local/bin/brassite --config=/config.yml
```

Or if you prefer Docker Compose:
```yaml
services:
  brassite:
    image: ghcr.io/teknologi-umum/brassite:edge
    command: "/usr/local/bin/brassite --config=/config.yml"
    volumes:
      - ./config.yml:/config.yml
    restart: on-failure:10
```

## License

```
Copyright 2024 Teknologi Umum <opensource@teknologiumum.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

See [LICENSE](./LICENSE)