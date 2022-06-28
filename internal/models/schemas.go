package models

import (
	"github.com/SamuelTissot/sqltime"
)

// unfortunately we end up with the duplication of fields in models (instead of embedding structs) due to bugs with gorm handling embedded structs
type Schema struct {
	ID          uint   `gorm:"primarykey"`
	ApiId       string `json:"api_id"`
	VersionId   string `json:"version_id"`
	RevisionId  string `json:"revision_id"`
	Filename    string `json:"filename"`
	Description string `json:"description"`
	// needing to use a third pary sqltime.Time package to avoid a bug in gorm where it want handle time.Time resolution correctly
	CreatedAt sqltime.Time `json:"created_at"`
}
