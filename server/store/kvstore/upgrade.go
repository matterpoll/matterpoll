package kvstore

import (
	"fmt"

	"github.com/blang/semver/v4"
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
