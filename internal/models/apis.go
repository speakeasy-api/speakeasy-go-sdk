package models

import (
	"github.com/SamuelTissot/sqltime"
)

// Api is the storage-side representation of an API.
type Api struct {
	ID          uint         `gorm:"primarykey"`
	WorkspaceId string       `json: "workspace_id"` // Uniquely identifies the workspace this api belongs to.
	Method      string       `json: "method"`
	Path        string       `json: "path"`
	CreatedAt   sqltime.Time `json: "created_at"`
	UpdatedAt   sqltime.Time `json: "updated_at"`
	DisplayName string       `json: "name"`                  // A human-friendly name.
	Description string       `json: "description,omitempty"` // A detailed description.
}
