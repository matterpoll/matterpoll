package kvstore

import (
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

const (
	perPage = 50
)

type upgrade struct {
	toVersion   string
	upgradeFunc func(*Store) error
}

func getUpgrades() []*upgrade {
	return []*upgrade{
		{toVersion: "1.1.0", upgradeFunc: nil},
		{toVersion: "1.2.0", upgradeFunc: nil},
		{toVersion: "1.3.0", upgradeFunc: nil},
		{toVersion: "1.4.0", upgradeFunc: upgradeTo14},
		{toVersion: "1.5.0", upgradeFunc: nil},
		{toVersion: "1.6.0", upgradeFunc: nil},
		{toVersion: "1.6.1", upgradeFunc: nil},
		{toVersion: "1.7.0", upgradeFunc: nil},
		{toVersion: "1.7.1", upgradeFunc: nil},
		{toVersion: "1.7.2", upgradeFunc: upgradeTo17_2},
		{toVersion: "1.8.0", upgradeFunc: upgradeTo18},
	}
}

// UpdateDatabase upgrades the database schema from a given version to the newest version.
func (s *Store) UpdateDatabase(pluginVersion string) error {
	currentVersion, err := s.System().GetVersion()
	if err != nil {
		return err
	}

	// If no version is set, set to to the newest version
	if currentVersion == "" {
		newestVersion := semver.MustParse(pluginVersion)
		// Don't store patch versions
		newestVersion.Patch = 0

		s.api.LogWarn(fmt.Sprintf("This looks to be a fresh install. Setting database schema version to %v.", newestVersion.String()))
		return s.System().SaveVersion(newestVersion.String())
	}

	for _, upgrade := range s.upgrades {
		if s.shouldPerformUpgrade(semver.MustParse(currentVersion), semver.MustParse(upgrade.toVersion)) {
			if upgrade.upgradeFunc != nil {
				err = upgrade.upgradeFunc(s)
				if err != nil {
					return err
				}
			}

			if err := s.System().SaveVersion(upgrade.toVersion); err != nil {
				return err
			}

			s.api.LogWarn(fmt.Sprintf("Update to version %v complete", upgrade.toVersion))
			currentVersion = upgrade.toVersion
		}
	}
	return nil
}

func (s *Store) shouldPerformUpgrade(currentSchemaVersion, expectedSchemaVersion semver.Version) bool {
	if currentSchemaVersion.LT(expectedSchemaVersion) {
		s.api.LogWarn(fmt.Sprintf("The database schema version of %v appears to be out of date.", currentSchemaVersion.String()))
		s.api.LogWarn(fmt.Sprintf("Attempting to upgrade the database schema version to %v.", expectedSchemaVersion.String()))
		return true
	}
	return false
}

func collectPollKeys(s *Store) ([]string, error) {
	var allKeys []string
	i := 0
	for {
		keys, appErr := s.api.KVList(i, perPage)
		if appErr != nil {
			return nil, errors.Wrap(appErr, "failed to list poll keys")
		}

		allKeys = append(allKeys, keys...)

		if len(keys) < perPage {
			break
		}

		i++
	}

	return allKeys, nil
}

func upgradeTo14(s *Store) error {
	allKeys, err := collectPollKeys(s)
	if err != nil {
		return err
	}

	for _, k := range allKeys {
		// Only migrate plugin keys
		if strings.HasPrefix(k, pollPrefix) {
			k = strings.TrimPrefix(k, pollPrefix)

			poll, err := s.Poll().Get(k)
			if err != nil {
				s.api.LogError("Failed to get poll for migration", "error", err.Error(), "pollID", k)
				continue
			}

			if poll.Settings.MaxVotes > 0 {
				// Already migrated
				continue
			}

			poll.Settings.MaxVotes = 1
			err = s.Poll().Save(poll)
			if err != nil {
				s.api.LogError("Failed to save poll after migration", "error", err.Error(), "pollID", k)
				continue
			}
		}
	}

	return nil
}

// upgradeTo17_2 convert existing polls to the new format that includes `Settings.AnonymousCreator` setting.
//
// New setting `AnonymousCreator“ without `omitempty` introduced in v1.7.0 causes the atomic transaction
// to fail when saving a poll. Additionally, just adding `omitempty` to Settings.AnonymousCreator introduced
// in v1.7.1 will also result in atomic transactions failure for poll with AnonymousCreator=false, which is
// created with Matterpoll v1.7.0.
// => see https://github.com/matterpoll/matterpoll/issues/562
func upgradeTo17_2(s *Store) error {
	allKeys, err := collectPollKeys(s)
	if err != nil {
		return err
	}

	for _, k := range allKeys {
		// Only migrate plugin keys
		if strings.HasPrefix(k, pollPrefix) {
			k = strings.TrimPrefix(k, pollPrefix)

			// poll is migrated when reading data
			poll, err := s.Poll().Get(k)
			if err != nil {
				s.api.LogError("Failed to get poll for migration", "error", err.Error(), "pollID", k)
				continue
			}

			err = s.Poll().Save(poll)
			if err != nil {
				s.api.LogError("Failed to save poll after migration", "error", err.Error(), "pollID", k)
				continue
			}
		}
	}

	return nil
}

func upgradeTo18(s *Store) error {
	allKeys, err := collectPollKeys(s)
	if err != nil {
		return err
	}

	for _, k := range allKeys {
		// Only migrate plugin keys
		if strings.HasPrefix(k, pollPrefix) {
			k = strings.TrimPrefix(k, pollPrefix)
			poll, err := s.Poll().Get(k)
			if err != nil {
				s.api.LogError("Failed to get poll for migration", "error", err.Error(), "pollID", k)
				continue
			}

			post, appErr := s.api.GetPost(poll.PostID)
			if appErr != nil {
				s.api.LogError("Failed to get post for migration", "error", appErr.Error(), "pollID", k, "postID", poll.PostID)
				continue
			}

			newAttachments := []*model.SlackAttachment{}
			for _, attachment := range post.Attachments() {
				newActions := []*model.PostAction{}
				for _, action := range attachment.Actions {
					if action.Type == "custom_matterpoll_admin_button" {
						action.Type = model.PostActionTypeButton
					}
					newActions = append(newActions, action)
				}
				attachment.Actions = newActions
				newAttachments = append(newAttachments, attachment)
			}
			model.ParseSlackAttachment(post, newAttachments)

			_, appErr = s.api.UpdatePost(post)
			if appErr != nil {
				s.api.LogError("Failed to update post after migration", "error", appErr.Error(), "pollID", k, "postID", poll.PostID)
				continue
			}
		}
	}

	return nil
}
