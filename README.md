# ![Matterpoll Logo](assets/logo.svg)

[![Build Status](https://img.shields.io/travis/com/matterpoll/matterpoll/master.svg)](https://travis-ci.com/matterpoll/matterpoll)
[![Code Coverage](https://img.shields.io/codecov/c/github/matterpoll/matterpoll/master.svg)](https://codecov.io/gh/matterpoll/matterpoll/branch/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/matterpoll/matterpoll)](https://goreportcard.com/report/github.com/matterpoll/matterpoll)
[![Releases](https://img.shields.io/github/release/matterpoll/matterpoll/all.svg)](https://github.com/matterpoll/matterpoll/releases/latest)


Matterpoll is a plugin for [Mattermost](https://mattermost.com/). It allows users to create poll by using a slash command.

Supported Mattermost Server Versions: 5.3+

## Installation

1. Go to the [releases page of this Github repository](https://github.com/matterpoll/matterpoll/releases/latest) and download the latest release for your Mattermost server.
2. Upload this file in the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
3. You should set **Enable integrations to override usernames** and **Enable integrations to override profile picture icons** in **System Console > Custom Integrations** to `true`.

## Usage

`/poll "Is Matterpoll great?"` creates a poll with the answer options "Yes" and "No". You can also leave out the double quotes and just type `/poll Is Matterpoll great?`.

If you want to define all answer options by yourself, type `/poll "Is Matterpoll great?" "Of course" "In any case" "Definitely"`- Note that the double quotes are required in this case.

## Troubleshooting

#### Pressing the poll buttons does nothing and creates a 400 error in the Mattermost log

Make sure to set your [Site URL](https://docs.mattermost.com/administration/config-settings.html?highlight=site%20url#site-url) properly.
For example, this error happens in case you set SiteURL starting with `http://`, in spite of running Mattermost server through https.
