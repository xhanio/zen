package api

import "github.com/xhanio/zen/pkg/types/entity"

type ReferenceSpec struct {
	SelectionText string `json:"selection_text" validate:"required,min=1,max=5000"`
}

type CreateCardRequest struct {
	Title                string         `json:"title" validate:"required,max=200"`
	Summary              *string        `json:"summary,omitempty" validate:"omitempty,max=500"`
	Content              string         `json:"content" validate:"max=1048576"`
	Format               *string        `json:"format,omitempty" validate:"omitempty,oneof=markdown html text"`
	LevelEntryID         *string        `json:"level_entry_id,omitempty" validate:"omitempty,len=26"`
	Genesis              *string        `json:"genesis,omitempty" validate:"omitempty,max=2000"`
	GroupID              string         `json:"group_id" validate:"required,len=26"`
	Tags                 []string       `json:"tags" validate:"omitempty,dive,max=50"`
	ParentCardID         *string        `json:"parent_card_id" validate:"omitempty,len=26"`
	SourceConversationID *string        `json:"source_conversation_id" validate:"omitempty,len=26"`
	Reference            *ReferenceSpec `json:"reference,omitempty" validate:"omitempty"`
}

type UpdateCardRequest struct {
	Title           *string   `json:"title" validate:"omitempty,max=200"`
	Summary         *string   `json:"summary,omitempty" validate:"omitempty,max=500"`
	Content         *string   `json:"content" validate:"omitempty,max=1048576"`
	Format          *string   `json:"format,omitempty" validate:"omitempty,oneof=markdown html text"`
	LevelEntryID    *string   `json:"level_entry_id,omitempty" validate:"omitempty,len=26"`
	ClearLevelEntry bool      `json:"clear_level_entry,omitempty"`
	Genesis         *string   `json:"genesis,omitempty" validate:"omitempty,max=2000"`
	GroupID         *string   `json:"group_id" validate:"omitempty,len=26"`
	Position        *int      `json:"position"`
	Tags            *[]string `json:"tags,omitempty" validate:"omitempty,dive,max=50"`
}

type ReorderCardRequest struct {
	Position int `json:"position" validate:"gte=0"`
}

type ReviewCardRequest struct {
	Grade string `json:"grade" validate:"required,oneof=LGTM DIGESTED GRILLED"`
}

type CardResponse struct {
	*entity.Card
}

type CardSpec struct {
	Title        string   `json:"title" validate:"required,max=200"`
	Summary      *string  `json:"summary,omitempty" validate:"omitempty,max=500"`
	Content      string   `json:"content" validate:"max=1048576"`
	Format       *string  `json:"format,omitempty" validate:"omitempty,oneof=markdown html text"`
	LevelEntryID *string  `json:"level_entry_id,omitempty" validate:"omitempty,len=26"`
	Genesis      *string  `json:"genesis,omitempty" validate:"omitempty,max=2000"`
	GroupID      *string  `json:"group_id,omitempty" validate:"omitempty,len=26"`
	Tags         []string `json:"tags,omitempty" validate:"omitempty,dive,max=50"`
	Position     *int     `json:"position,omitempty"`
}

type DecomposeRequest struct {
	ParentCardID     string     `json:"parent_card_id" validate:"required,len=26"`
	ContainerContent *string    `json:"container_content,omitempty" validate:"omitempty,max=1048576"`
	Cards            []CardSpec `json:"cards" validate:"required,min=1,dive"`
}

type DecomposeResponse struct {
	Cards []*entity.Card `json:"cards"`
}

type ComposeRequest struct {
	SourceCardIDs []string `json:"source_card_ids" validate:"required,min=2,unique,dive,len=26"`
	Target        CardSpec `json:"target" validate:"required"`
}

type ComposeResponse struct {
	Card *entity.Card `json:"card"`
}

type TrashResponse struct {
	Cards []*entity.Card `json:"cards"`
}

type EmptyTrashResponse struct {
	Purged int `json:"purged"`
}
