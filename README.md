# ![Matterpoll Logo](assets/logo.svg)

[![Build Status](https://img.shields.io/travis/com/matterpoll/matterpoll.svg)](https://travis-ci.com/matterpoll/matterpoll)
[![Code Coverage](https://img.shields.io/codecov/c/github/matterpoll/matterpoll.svg)](https://codecov.io/gh/matterpoll/matterpoll/branch/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/matterpoll/matterpoll)](https://goreportcard.com/report/github.com/matterpoll/matterpoll)
[![Releases](https://img.shields.io/github/release/matterpoll/matterpoll.svg)](https://github.com/matterpoll/matterpoll/releases/latest)


MatterPoll is a plugin for [Mattermost](https://mattermost.com/). It allows users to create poll by using a slash command.

Supported Mattermost Server Versions: 5.2+

## Installation

1. Go to the [releases page of this Github repository](https://github.com/matterpoll/matterpoll/releases/latest) and download the latest release for your Mattermost server.
2. Upload this file in the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
3. Add `localhost` to `AllowedUntrustedInternalConnections` in `config.json`.
4. You should set `EnablePostUsernameOverride` and `EnablePostIconOverride` to `true`.

## Usage

`/matterpoll "Is MatterPoll great?"` creates a poll with the answer options "Yes" and "No". You can also leave out the double quotes and just type `/matterpoll Is MatterPoll great?`.

If you want to define all answer options by yourself, type `/matterpoll "Is MatterPoll great?" "Of course" "In any case" "Definitely"`- Note that the double quotes are required in this case.
