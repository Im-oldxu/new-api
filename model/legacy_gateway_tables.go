package model

import (
	"fmt"

	"gorm.io/gorm"
)

var legacyGatewayTables = []struct {
	source  string
	archive string
}{
	{source: "abilities", archive: "legacy_abilities"},
	{source: "channels", archive: "legacy_channels"},
	{source: "models", archive: "legacy_models"},
	{source: "vendors", archive: "legacy_vendors"},
}

// archiveLegacyGatewayTables preserves existing gateway data while removing
// its tables from the active schema. It preflights every destination before
// renaming and compensates earlier renames if a later DDL statement fails.
// This is safe to retry and never overwrites an existing archive table.
func archiveLegacyGatewayTables(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database is required to archive legacy gateway tables")
	}

	pending := make([]struct {
		source  string
		archive string
	}, 0, len(legacyGatewayTables))
	for _, table := range legacyGatewayTables {
		hasSource := db.Migrator().HasTable(table.source)
		hasArchive := db.Migrator().HasTable(table.archive)
		if hasSource && hasArchive {
			return fmt.Errorf("cannot archive %s: destination table %s already exists", table.source, table.archive)
		}
		if hasSource {
			pending = append(pending, table)
		}
	}

	renamed := make([]struct {
		source  string
		archive string
	}, 0, len(pending))
	for _, table := range pending {
		if err := db.Migrator().RenameTable(table.source, table.archive); err != nil {
			var rollbackFailures []string
			for i := len(renamed) - 1; i >= 0; i-- {
				previous := renamed[i]
				if rollbackErr := db.Migrator().RenameTable(previous.archive, previous.source); rollbackErr != nil {
					rollbackFailures = append(rollbackFailures, fmt.Sprintf("%s: %v", previous.archive, rollbackErr))
				}
			}
			if len(rollbackFailures) > 0 {
				return fmt.Errorf("archive legacy table %s: %w; rollback failures: %v", table.source, err, rollbackFailures)
			}
			return fmt.Errorf("archive legacy table %s: %w", table.source, err)
		}
		renamed = append(renamed, table)
	}
	return nil
}
