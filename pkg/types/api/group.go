package api

import "github.com/xhanio/zen/pkg/types/entity"

type CreateGroupRequest struct {
	Name         string              `json:"name" validate:"required,max=100"`
	Rule         string              `json:"rule,omitempty" validate:"omitempty,max=4000"`
	LevelCatalog []entity.LevelEntry `json:"level_catalog,omitempty"`
}

type UpdateGroupRequest struct {
	Name         *string              `json:"name" validate:"omitempty,max=100"`
	Rule         *string              `json:"rule,omitempty" validate:"omitempty,max=4000"`
	Position     *int                 `json:"position"`
	LevelCatalog *[]entity.LevelEntry `json:"level_catalog,omitempty"`
}

type GroupResponse struct {
	*entity.Group
}
