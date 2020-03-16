# ![Matterpoll Logo](images/logo.svg)

[![Build Status](https://img.shields.io/circleci/project/github/matterpoll/matterpoll/master.svg)](https://circleci.com/gh/matterpoll/matterpoll)
[![Code Coverage](https://img.shields.io/codecov/c/github/matterpoll/matterpoll/master.svg)](https://codecov.io/gh/matterpoll/matterpoll/branch/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/matterpoll/matterpoll)](https://goreportcard.com/report/github.com/matterpoll/matterpoll)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/2588/badge)](https://bestpractices.coreinfrastructure.org/projects/2588)
[![Releases](https://img.shields.io/github/release/matterpoll/matterpoll.svg)](https://github.com/matterpoll/matterpoll/releases/latest)

Matterpoll is a plugin for [Mattermost](https://mattermost.com/). It allows users to create poll by using a slash command.


## Installation

1. Go to the [releases page of this GitHub repository](https://github.com/matterpoll/matterpoll/releases/latest) and download the latest release for your Mattermost server.
2. Upload this file in the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
3. You should set **Enable integrations to override usernames** and **Enable integrations to override profile picture icons** in **System Console > Custom Integrations** to `true`.


## Settings
You can configure Matterpoll from **System Console > Plugins > Matterpoll**.

* **Trigger Word**: Change trigger word for poll command. (default `/poll`)
* **Experimental UI**: Enable new experimental UI for a poll post. 

## Usage

`/poll "Is Matterpoll great?"` creates a poll with the answer options "Yes" and "No". You can also leave out the double quotes and just type `/poll Is Matterpoll great?`.

If you want to define all answer options by yourself, type `/poll "Is Matterpoll great?" "Of course" "In any case" "Definitely"`- Note that the double quotes are required in this case.

### Poll Settings

Poll Settings provider further customisation, e.g. `/poll "Is Matterpoll great?" "Of course" "In any case" "Definitely" --progress --anonymous`. The available Poll Settings are:
- `--anonymous`: Don't show who voted for what at the end
- `--progress`: During the poll, show how many votes each answer option got
- `--public-add-option`: Allow all users to add additional options


## Localization

Matterpoll supports localization of user specify messages. You can change language of poll message by setting it in **System Console > General > Localization > Default Server Language**. Language of messages that only a user can see (e.g.: help messages, error messages) use the language set in **Account Settings > Display > Language**.

The currently supported languages are:
- English
- France
- German
- Japanese
- Polish
- Spanish


## Troubleshooting

#### Pressing the poll buttons does nothing and creates a 400 error in the Mattermost log

Make sure to set your [Site URL](https://docs.mattermost.com/administration/config-settings.html?highlight=site%20url#site-url) properly.
For example, this error happens in case you set SiteURL starting with `http://`, in spite of running Mattermost server through https.


## Contributing

We welcome contributions for bug reports, issues, feature requests, feature implementations and pull requests. Feel free to [**file a new issue**](https://github.com/matterpoll/matterpoll/issues/new/choose) or join the [**Matterpoll channel**](https://community.mattermost.com/core/channels/matterpoll) on the Mattermost community server.

For a complete guide on contributing to Matterpoll, see the [Contribution Guideline](CONTRIBUTING.md).
