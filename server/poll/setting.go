package poll

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/matterpoll/matterpoll/server/utils"
)

var votesSettingPattern = regexp.MustCompile(`^votes=(\d+)$`)
var endSettingPattern = regexp.MustCompile(`^end=(.+)$`)

type Settings struct {
	Anonymous        bool
	AnonymousCreator bool
	Progress         bool
	PublicAddOption  bool
	End              *time.Time
	MaxVotes         int `json:"max_votes"`
}

func (s Settings) String() string {
	var settingsText []string
	if s.Anonymous {
		settingsText = append(settingsText, "anonymous")
	}
	if s.AnonymousCreator {
		settingsText = append(settingsText, "anonymous-creator")
	}
	if s.Progress {
		settingsText = append(settingsText, "progress")
	}
	if s.PublicAddOption {
		settingsText = append(settingsText, "public-add-option")
	}
	if s.End != nil {
		settingsText = append(settingsText, fmt.Sprintf("ends at %s", s.End.Local().Format(time.DateTime)))
	}
	if s.MaxVotes > 1 {
		settingsText = append(settingsText, fmt.Sprintf("votes=%d", s.MaxVotes))
	}

	return strings.Join(settingsText, ", ")
}

// NewSettingsFromStrings creates a new settings with the given parameter.
func NewSettingsFromStrings(strs []string) (Settings, *utils.ErrorMessage) {
	settings := Settings{MaxVotes: 1}
	for _, str := range strs {
		switch {
		case str == SettingKeyAnonymous:
			settings.Anonymous = true
		case str == SettingKeyAnonymousCreator:
			settings.AnonymousCreator = true
		case str == SettingKeyProgress:
			settings.Progress = true
		case str == SettingKeyPublicAddOption:
			settings.PublicAddOption = true
		case endSettingPattern.MatchString(str):
			end, errMsg := parseEndSettings(str)
			if errMsg != nil {
				return settings, errMsg
			}
			settings.End = &end
		case votesSettingPattern.MatchString(str):
			i, errMsg := parseVotesSettings(str)
			if errMsg != nil {
				return settings, errMsg
			}
			settings.MaxVotes = i
		default:
			return settings, &utils.ErrorMessage{
				Message: &i18n.Message{
					ID:    "poll.newPoll.unrecognizedSetting",
					Other: "Unrecognized poll setting: {{.Setting}}",
				},
				Data: map[string]interface{}{
					"Setting": str,
				},
			}
		}
	}
	return settings, nil
}

// NewSettingsFromSubmission creates a new settings with the given parameter.
func NewSettingsFromSubmission(submission map[string]interface{}) (Settings, *utils.ErrorMessage) {
	settings := Settings{MaxVotes: 1}
	for k, v := range submission {
		switch {
		case k == "setting-multi":
			f, ok := v.(float64)
			if ok {
				settings.MaxVotes = int(f)
			}
		case k == "setting-end":
			end, err := parseDateOrDuration(v.(string))
			if err != nil {
				return settings, err
			}
			settings.End = &end
		case strings.HasPrefix(k, "setting-"):
			b, ok := v.(bool)
			if b && ok {
				s := strings.TrimPrefix(k, "setting-")
				switch s {
				case SettingKeyAnonymous:
					settings.Anonymous = true
				case SettingKeyAnonymousCreator:
					settings.AnonymousCreator = true
				case SettingKeyProgress:
					settings.Progress = true
				case SettingKeyPublicAddOption:
					settings.PublicAddOption = true
				}
			}
		}
	}
	return settings, nil
}

// parseVotesSettings parses setting for votes ("--votes=X")
func parseVotesSettings(s string) (int, *utils.ErrorMessage) {
	e := votesSettingPattern.FindStringSubmatch(s)
	if len(e) != 2 {
		return 0, getUnexpectedErrorMessage("poll.newPoll.votesettings.unexpectedError", s)
	}
	i, err := strconv.Atoi(e[1])
	if err != nil {
		return 0, getUnexpectedErrorMessage("poll.newPoll.votesettings.invalidSetting", s)
	}
	return i, nil
}

// parseEndSettings parses setting for end date ("--end=X")
func parseEndSettings(s string) (time.Time, *utils.ErrorMessage) {
	e := endSettingPattern.FindStringSubmatch(s)
	if len(e) != 2 {
		return time.Time{}, getUnexpectedErrorMessage("poll.newPoll.endsettings.unexpectedError", s)
	}

	date, err := parseDateOrDuration(e[1])

	if err != nil {
		return time.Time{}, err
	}

	return date, nil
}

// parseDateOrDuration parses given string date or duration to time.Time
func parseDateOrDuration(value string) (time.Time, *utils.ErrorMessage) {
	var date time.Time

	if value == "tomorrow" {
		date = time.Now().Add(time.Hour * time.Duration(24)).UTC().Round(time.Second)
		return date, nil
	}

	duration, err := time.ParseDuration(value)
	if err == nil {
		date = time.Now().Add(duration).UTC().Round(time.Second)
	} else {
		date, err = parseDate(value)
	}

	if err != nil {
		return time.Time{}, getUnexpectedErrorMessage("poll.newPoll.endsettings.invalidSetting", value)
	}

	if date.Before(time.Now()) {
		return time.Time{}, &utils.ErrorMessage{
			Message: &i18n.Message{
				ID:    "poll.newPoll.endsettings.beforeNow",
				Other: "The end time {{.Date}} cannot be set to a time before the current time",
			},
			Data: map[string]interface{}{
				"Date": date.String(),
			},
		}
	}

	return date, nil
}

// parseDate try to parse given string date to time.Time using several layouts
func parseDate(value string) (time.Time, error) {
	date, err := time.Parse(EndSettingStandardLayout, value)
	if err == nil {
		_, offset := time.Now().Zone()
		date = date.Add(-time.Duration(offset) * time.Second).UTC()

		return date, nil
	}

	date, err = time.Parse(EndSettingSecondsLayout, value)
	if err == nil {
		_, offset := time.Now().Zone()
		date = date.Add(-time.Duration(offset) * time.Second).UTC()

		return date, nil
	}

	date, err = time.Parse(EndSettingTimezoneLayout, value)
	if err == nil {
		return date.UTC(), nil
	}

	date, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return date.UTC(), nil
	}

	return time.Time{}, err
}
