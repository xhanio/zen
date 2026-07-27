package api

import "github.com/xhanio/zen/pkg/types/entity"

type CreateReferenceRequest struct {
	SourceCardID   string `json:"source_card_id" validate:"required,len=26"`
	DerivedCardID  string `json:"derived_card_id" validate:"required,len=26"`
	ConversationID string `json:"conversation_id" validate:"required,len=26"`
	SelectionText  string `json:"selection_text" validate:"omitempty,max=5000"`
	// MessageID names the message whose selection this reference anchors to.
	// When set, the message's range AND its selection_text are copied onto
	// the reference and SelectionText above is ignored: the SPA captured that
	// copy verbatim at drag time, while a caller-supplied one is a retype.
	MessageID *string `json:"message_id,omitempty" validate:"omitempty,len=26"`
}

type ListReferencesRequest struct {
	SourceCardID   *string `json:"source_card_id,omitempty" query:"source_card_id" validate:"omitempty,len=26"`
	DerivedCardID  *string `json:"derived_card_id,omitempty" query:"derived_card_id" validate:"omitempty,len=26"`
	ConversationID *string `json:"conversation_id,omitempty" query:"conversation_id" validate:"omitempty,len=26"`
}

type ListReferencesResponse struct {
	References []*entity.Reference `json:"references"`
}
