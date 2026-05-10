package spaces

// Space is the domain model for a workspace space.
type Space struct {
	ID                    string  `json:"id"`
	Name                  string  `json:"name"`
	Slug                  string  `json:"slug"`
	Icon                  string  `json:"icon"`
	DefaultPermission     string  `json:"defaultPermission"`
	McpWriteEnabled       bool    `json:"mcpWriteEnabled"`
	SnapshotRetentionDays *int64  `json:"snapshotRetentionDays,omitempty"`
	CreatedAt             int64   `json:"createdAt"`
}

// CreateRequest is the body for POST /api/spaces.
type CreateRequest struct {
	Name                  string `json:"name"`
	Icon                  string `json:"icon,omitempty"`
	DefaultPermission     string `json:"defaultPermission,omitempty"`
	McpWriteEnabled       bool   `json:"mcpWriteEnabled,omitempty"`
	SnapshotRetentionDays *int64 `json:"snapshotRetentionDays,omitempty"`
}

// UpdateRequest is the body for PATCH /api/spaces/{id}.
type UpdateRequest struct {
	Name                  string `json:"name,omitempty"`
	Icon                  string `json:"icon,omitempty"`
	DefaultPermission     string `json:"defaultPermission,omitempty"`
	McpWriteEnabled       *bool  `json:"mcpWriteEnabled,omitempty"`
	SnapshotRetentionDays *int64 `json:"snapshotRetentionDays,omitempty"`
}

// ListResponse is returned from GET /api/spaces.
type ListResponse struct {
	Spaces []Space `json:"spaces"`
}
