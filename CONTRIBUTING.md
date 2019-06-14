# Contributing to Matterpoll

Thank you for your interest in contributing! Join the join the [**Matterpoll channel**](https://community.mattermost.com/core/channels/matterpoll) on the Mattermost community server for discussion about this plugin.


## Reporting Issues

If you think you found a bug in Hugo, [please use the GitHub issue tracker](https://github.com/matterpoll/matterpoll/issues/new?template=bug_report.md) to open an issue. When opening an issue, please provide the required information in the issue template.


## Helping with translations

Matterpoll supports localization in various languages. Because we as the maintainer only speak a small amount of language, we rely on contributors to help us with the translations.

The translations process is:
- While developing, new translation messages may be added or existing ones changed.
- When a new version will releases soon, a maintainer will open an issue informing about this.
- The maintainer will ping all translation maintainer to inform them about this.
- They open PR's with new translations, which (may) get reviewed by other translators.
- After all translation PR's are merged, the new version is released.

### Translation Maintainer

- France: [@ldidry](https://github.com/ldidry)
- German: [@hanzei](https://github.com/hanzei)
- Japanese: [@kaakaa](https://github.com/kaakaa/)

### Translating new messages

To translate new or changed translation messages, you need to first ensure all translation messages are correctly extracted:

`goi18n extract -format json -outdir assets/i18n/ server/`

Then update your translation files:

`goi18n merge -format json -outdir assets/i18n/ assets/i18n/active.*.json`

Translate all messages in `asserts/i18n/translate.*.json` for the languages you are comfortable with.

Merge the translated messages into the active message files:

`goi18n merge -format json -outdir assets/i18n/ assets/i18n/active.*.json assets/i18n/translate.*.json`

Commit **only the language files you touched** and [submit a PR](https://github.com/matterpoll/matterpoll/compare).

### Translating a new language

Let's say you want to translate the local `de`. Replace  `de` in the following commands with the local you want to translate. See [here](https://github.com/mattermost/mattermost-server/tree/master/i18n) for the list of possible locals.

Create a translation file:

`touch asserts/i18n/translate.de.json`

Merge all current messages into your translation file:

`goi18n merge -format json -outdir assets/i18n/ assets/i18n/active.en.json assets/i18n/translate.de.json`

Translate all messages in `asserts/i18n/translate.de.json` and rename it to `active.de.json`.

[Submit a PR](https://github.com/matterpoll/matterpoll/compare) with this file and add you to the list of [Translation Maintainer](#translation-maintainer)


## Submitting Patches

If you contributing a feature, [please open a feature request](https://github.com/matterpoll/matterpoll/issues/new?template=feature_request.md) first. This way the feature can be discussed and fully specified before you start working on this. Small code changes can be submitted without opening an issue first.

You can find all issue that we seek help with [here](https://github.com/matterpoll/matterpoll/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22Help+Wanted%22).
