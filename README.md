# Memo

inspired by https://github.com/Dieterbe/anthracite/ but now
as part of your chatops, chatdev, chatmarketing, chatWhatever workflow!

Comes with 2 programs:

* memo-cli: submit grafana annotations from the cli
* memod: slack bot, so you can can submit annotations from slack

## Huh?

Turn a slack message like this ...
![usage in slack](./docs/img/memo-slack-screenshot.png)
... into an annotation like this:
![usage in slack](./docs/img/memo-in-grafana-from-slack.png)
Luckily somebody shared this memo on slack, otherwise somebody might freak out if they see this chart!

## memo-cli

```
Usage of ./memo-cli:
  -config string
    	config file location (default "~/.memo.toml")
  -msg string
    	message to submit
  -tags value
    	One or more comma-separated tags to submit, in addition to 'memo', 'user:<unix-username>' and 'host:<hostname>'
  -ts int
    	unix timestamp. always defaults to 'now' (default 1557953985)
```

## memod

Connects to slack and listens for "memo" messages which - if correctly formatted - will result in an annotation on the configured Grafana server

#### Message format

```
memo [timespec] <msg> [tags]
```

`[foo]` denotes that `foo` is optional.


#### timespec

defaults to `25`, so by default it assumes your message is about 25 seconds after the actual event happened.

It can have the following formats:

* `<duration>` like 0 (seconds), 10 (seconds), 30s, 1min20s, 2h, etc. see https://github.com/raintank/dur denotes how long ago the event took place
* `<RFC3339 spec>` like `2013-06-05T14:10:43Z`

#### msg

free-form text message, but if the first word looks like a timespec it will be interpreted as such.  Any words at the end with `:` in them will be interpreted as tags.

#### tags

default tags included:

* `memo`
* `chan:slack channel (if not a PM)`
* `author:slack username`

you can extend these. any words at the end of the command that have `:` will be used as key-value tags.
But you cannot override any of the default tags

# Installation

## Configure Slack (only for memod)

1. Create a [new slack app](https://api.slack.com/apps)
1. Go to OAuth & Permissions and enable the bot token scopes listed [below](#oauth-scopes-required)
1. Click "Install App" to then connect it to your workspace and generate an `xoxb-xxxxx` token, which is the `bot_token` in the `[slack]` section
1. Enable socket mode on your application, which will then generate an `xapp-xxxxx` token, which is the `app_token` in the `[slack]` section
1. Enable event subscriptions
1. Subscribe to bot events: `message.channels` and `message.im`

### OAuth scopes required:
- channels:history
- channels:read
- chat:write
- im:history
- im:read
- users:read

## Configure Grafana

1. Log into your Grafana instance, eg https://something.grafana.net
1. Click into Administration > Users and access > Service accounts
1. Create a service account with the roles; `Annotations:Writer`, `Annotations:Dashboard annotation writer` & `Annotations:Organization annotation writer`
1. Add a new service account token and store that in your config.toml under `api_key` in the `[grafana]` section

## Install the program

Currently we don't publish distribution packages, docker images etc.
So for now, you need to build the binary/binaries from source

First, [install golang](https://golang.org/dl/)
Then, run any of these commands to download the source code and build the binaries:

```
go get github.com/grafana/memo/cmd/memod    # only memod, the slack bot
go get github.com/grafana/memo/cmd/memo-cli # only memo-cli, the command line tool
go get github.com/grafana/memo/cmd/...      # both
```

You will then have the binaries in `$HOME/bin` or in `$GOPATH/bin` if you have a custom GOPATH set.

## config file for memo-cli

Put this file in `~/.memo.toml`

```
[grafana]
api_key = "<grafana api key, editor role>"
api_url = "https://<grafana host>/api/"
```

## config file for memod

Put a config file like below in `/etc/memo.toml`.

```
# one of trace debug info warn error fatal panic
log_level = "info"

[slack]
enabled = true
bot_token = "<slack bot token>"
app_token = "<slack app token>"

[discord]
enabled = true
bot_token = "<discord bot token>"

[grafana]
api_key = "<grafana api key, editor role>"
api_url = "http://localhost/api/"
```

## auto-starting memod

If you use upstart, you need to create an init file and put it in /etc/init/memo.conf
For your convenience you can use our [example upstart config file](./var/upstart-memo.conf)
In this case also copy the binary to `/usr/bin/memod`.

## Set up the Grafana integration

You need to create a new annotation query on your applicable dashboards.
Make sure to set it to the Grafana datasource and use filtering by tag, you can use tags like `memo` and `chan:<chan-name>` or any other tags of your choosing.

![Grafana annotation query](./docs/img/configure-grafana-for-memo.png)

# Docker

A docker image is compiled for convenience:

```
# memod
docker run -v "${PWD}/config.toml:/etc/memo.toml" ghcr.io/grafana/memo:master

# memo-cli
docker run -v "${PWD}/config.toml:/etc/memo.toml" ghcr.io/grafana/memo:master memo-cli -config /etc/memo.toml -msg "test"
```
