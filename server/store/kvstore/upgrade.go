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

type migrationResults struct {
	processed int
	skipped   int
	failed    int
}

func (ms *migrationResults) String() string {
	return fmt.Sprintf("processed: %d, skipped: %d, failed: %d", ms.processed, ms.skipped, ms.failed)
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

func (s *Store) applyUpgradeFunc(migrateFunc func(key string) error) error {
	i := 0
	for {
		keys, appErr := s.api.KVList(i, perPage)
		if appErr != nil {
			return errors.Wrap(appErr, "failed to list poll keys")
		}

		for _, k := range keys {
			// Migrate only plugin keys
			if !strings.HasPrefix(k, pollPrefix) {
				continue
			}
			pollID := strings.TrimPrefix(k, pollPrefix)
			if err := migrateFunc(pollID); err != nil {
				s.api.LogWarn("Failed to apply upgrade function", "poll_id", k, "error", err.Error())
			}
		}

		if len(keys) < perPage {
			break
		}

		i++
	}
	return nil
}

func upgradeTo14(s *Store) error {
	status := new(migrationResults)
	err := s.applyUpgradeFunc(func(pollId string) error {
		poll, err := s.Poll().Get(pollId)
		if err != nil {
			status.failed++
			return errors.Wrap(err, "Failed to get poll for migration")
		}

		if poll.Settings.MaxVotes > 0 {
			// Already migrated
			status.skipped++
			return nil
		}

		poll.Settings.MaxVotes = 1
		err = s.Poll().Save(poll)
		if err != nil {
			status.failed++
			return errors.Wrap(err, "Failed to save poll after migration")
		}

		status.processed++
		return nil
	})
	s.api.LogInfo("Migration to v1.4.0 completed", "results", status.String())
	return err
}

// upgradeTo17_2 convert existing polls to the new format that includes `Settings.AnonymousCreator` setting.
//
// New setting `AnonymousCreator“ without `omitempty` introduced in v1.7.0 causes the atomic transaction
// to fail when saving a poll. Additionally, just adding `omitempty` to Settings.AnonymousCreator introduced
// in v1.7.1 will also result in atomic transactions failure for poll with AnonymousCreator=false, which is
// created with Matterpoll v1.7.0.
// => see https://github.com/matterpoll/matterpoll/issues/562
func upgradeTo17_2(s *Store) error {
	status := new(migrationResults)
	err := s.applyUpgradeFunc(func(pollId string) error {
		// poll is migrated when reading data
		poll, err := s.Poll().Get(pollId)
		if err != nil {
			status.failed++
			return errors.Wrap(err, "Failed to get poll for migration")
		}

		err = s.Poll().Save(poll)
		if err != nil {
			status.failed++
			return errors.Wrap(err, "Failed to save poll after migration")
		}

		status.processed++
		return nil
	})
	s.api.LogInfo("Migration to v1.7.2 completed", "results", status.String())
	return err
}

// upgradeTo18 migrates the poll post attachments to avoid using custom actions types
// for upcoming Mattermost's new validation schema.
func upgradeTo18(s *Store) error {
	status := new(migrationResults)
	err := s.applyUpgradeFunc(func(pollId string) error {
		poll, err := s.Poll().Get(pollId)
		if err != nil {
			status.failed++
			return errors.Wrap(err, "Failed to get poll for migration")
		}
		post, appErr := s.api.GetPost(poll.PostID)
		if appErr != nil {
			status.failed++
			return errors.Wrap(appErr, "Failed to get post for migration")
		}

		toMigrate := false
		attachments := post.Attachments()
		for _, attachment := range attachments {
			for _, action := range attachment.Actions {
				if action.Type == "custom_matterpoll_admin_button" {
					toMigrate = true
					action.Type = model.PostActionTypeButton
				}
			}
		}
		if !toMigrate {
			status.skipped++
			return nil
		}

		model.ParseSlackAttachment(post, attachments)
		_, appErr = s.api.UpdatePost(post)
		if appErr != nil {
			status.failed++
			return errors.Wrap(appErr, "Failed to update post after migration")
		}
		status.processed++
		return nil
	})
	s.api.LogInfo("Migration to v1.8.0 completed", "results", status.String())
	return err
}
