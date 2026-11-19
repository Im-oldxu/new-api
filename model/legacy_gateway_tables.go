package model

import (
	"fmt"

	"gorm.io/gorm"
)

var legacyGatewayTables = []struct {
	source string
	archive string
}{
	{source: "abilities", archive: "legacy_abilities"},
	{source: "channels", archive: "legacy_channels"},
	{source: "models", archive: "legacy_models"},
	{source: "vendors", archive: "legacy_vendors"},
}

// archiveLegacyGatewayTables preserves existing gateway data while removing
// its tables from the active schema. The operation is transactional and
// idempotent; a name collision fails closed instead of overwriting data.
func archiveLegacyGatewayTables(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database is required to archive legacy gateway tables")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, table := range legacyGatewayTables {
			migrator := tx.Migrator()
			if !migrator.HasTable(table.source) {
				continue
			}
			if migrator.HasTable(table.archive) {
				return fmt.Errorf("cannot archive %s: destination table %s already exists", table.source, table.archive)
			}
			if err := migrator.RenameTable(table.source, table.archive); err != nil {
				return fmt.Errorf("archive legacy table %s: %w", table.source, err)
			}
		}
		return nil
	})
}
