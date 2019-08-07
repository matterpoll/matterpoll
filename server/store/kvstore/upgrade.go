package kvstore

import (
	"fmt"

	"github.com/blang/semver"
)

type upgrade struct {
	toVersion   string
	upgradeFunc func(*Store) error
}

func getUpgrades() []*upgrade {
	return []*upgrade{
		{toVersion: "1.1.0", upgradeFunc: nil},
	}
}

// UpdateDatabase upgrades the database schema from a given version to the newest version.
func (s *Store) UpdateDatabase(pluginVersion string) error {
	v, err := s.System().GetVersion()
	if err != nil {
		return err
	}

	// If no version is set, set to to the newest version
	if v == "" {
		newestSchema := semver.MustParse(pluginVersion)
		// Don't store patch versions
		newestSchema.Patch = 0

		s.api.LogWarn(fmt.Sprintf("This looks to be a fresh install. Setting database schema version to %v.", newestSchema.String()))
		return s.System().SaveVersion(newestSchema.String())
	}

	for _, upgrade := range s.upgrades {
		v, err := s.System().GetVersion()
		if err != nil {
			return err
		}

		currentSchemaVersion := semver.MustParse(v)
		if s.shouldPerformUpgrade(currentSchemaVersion, semver.MustParse(upgrade.toVersion)) {
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
		}
	}
	return nil
}

func (s *Store) shouldPerformUpgrade(currentSchemaVersion semver.Version, expectedSchemaVersion semver.Version) bool {
	if currentSchemaVersion.LT(expectedSchemaVersion) {
		s.api.LogWarn(fmt.Sprintf("The database schema version of %v appears to be out of date.", currentSchemaVersion.String()))
		s.api.LogWarn(fmt.Sprintf("Attempting to upgrade the database schema version to %v.", expectedSchemaVersion.String()))
		return true
	}
	return false
}
