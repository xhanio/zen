package api

import "github.com/xhanio/zen/pkg/types/entity"

type RenameTagRequest struct {
	NewName string `json:"new_name" validate:"required,max=50"`
}

type TagResponse struct {
	*entity.Tag
}
