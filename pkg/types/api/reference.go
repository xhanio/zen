package api

import "github.com/xhanio/zen/pkg/types/entity"

type CreateReferenceRequest struct {
	SourceCardID   string `json:"source_card_id" validate:"required,len=26"`
	DerivedCardID  string `json:"derived_card_id" validate:"required,len=26"`
	ConversationID string `json:"conversation_id" validate:"required,len=26"`
	SelectionText  string `json:"selection_text" validate:"required,min=1,max=5000"`
}

type ListReferencesRequest struct {
	SourceCardID   *string `json:"source_card_id,omitempty" query:"source_card_id" validate:"omitempty,len=26"`
	DerivedCardID  *string `json:"derived_card_id,omitempty" query:"derived_card_id" validate:"omitempty,len=26"`
	ConversationID *string `json:"conversation_id,omitempty" query:"conversation_id" validate:"omitempty,len=26"`
}

type ListReferencesResponse struct {
	References []*entity.Reference `json:"references"`
}
