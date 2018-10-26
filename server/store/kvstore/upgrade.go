package kvstore

import (
	"fmt"

	"github.com/blang/semver"
)

func (s *Store) UpdateDatabase() error {
	v, err := s.System().GetVersion()
	if err != nil {
		return err
	}
	// If no version is set, set to to the newest version
	if v == "" {
		// TODO: Dont hardcode this
		v = "1.0.0"
	}

	// TODO: Uncomment following condition when version 1.1.0 is released
	/*
		currentSchemaVersion := semver.MustParse(v)
		if err := s.UpgradeDatabaseToVersion11(currentSchemaVersion); err != nil {
			return err
		}
	*/

	return nil
}

func (s *Store) shouldPerformUpgrade(currentSchemaVersion semver.Version, expectedSchemaVersion semver.Version) bool {
	if currentSchemaVersion.LT(expectedSchemaVersion) {
		s.api.LogWarn(fmt.Sprintf("The database schema version of %v appears to be out of date", currentSchemaVersion.String()))
		s.api.LogWarn(fmt.Sprintf("Attempting to upgrade the database schema version to %v", expectedSchemaVersion.String()))
		return true
	}
	return false
}

/*
func (s *Store) UpgradeDatabaseToVersion11(currentSchemaVersion semver.Version) error {
	if s.shouldPerformUpgrade(currentSchemaVersion, semver.MustParse("1.1.0")) {
		s.api.LogWarn("Update complete")
		// Do migration
		if err := s.System().SaveVersion("1.1.0"); err != nil {
			return err
		}
	}
	return nil
}
*/
