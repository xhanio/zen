package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

// groupRule returns the group's rule text, or "" if the group has none or
// cannot be read. Best-effort: the echo must never fail the card operation.
func (m *manager) groupRule(ctx context.Context, groupID string) string {
	if groupID == "" {
		return ""
	}
	g, err := m.backend.Group().Get(ctx, groupID)
	if err != nil || g == nil {
		return ""
	}
	return g.Rule
}

func (m *manager) registerTools() {
	m.registerGroupTools()
	m.registerTagTools()
	m.registerCardTools()
	m.registerConversationTools()
	m.registerSearchTools()
	m.registerDecomposeTool()
	m.registerComposeTool()
	m.registerReferenceTools()
}

// ---- Group ----

type groupCreateIn struct {
	Name         string              `json:"name"`
	Rule         string              `json:"rule,omitempty"`
	LevelCatalog []entity.LevelEntry `json:"level_catalog,omitempty"`
}
type groupCreateOut struct {
	Group *entity.Group `json:"group"`
}

type groupListOut struct {
	Groups []*entity.Group `json:"groups"`
}

type groupGetIn struct {
	ID string `json:"id"`
}
type groupGetOut struct {
	Group *entity.Group `json:"group"`
}

type groupUpdateIn struct {
	ID           string               `json:"id"`
	Name         *string              `json:"name,omitempty"`
	Rule         *string              `json:"rule,omitempty"`
	Position     *int                 `json:"position,omitempty"`
	LevelCatalog *[]entity.LevelEntry `json:"level_catalog,omitempty"`
}
type groupUpdateOut struct {
	Group *entity.Group `json:"group"`
}

type groupDeleteIn struct {
	ID        string `json:"id"`
	Recursive bool   `json:"recursive,omitempty"`
}
type groupDeleteOut struct {
	Deleted bool `json:"deleted"`
}

func (m *manager) registerGroupTools() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "group.create",
		Description: `Create a new group. Groups are flat (no nesting). "level_catalog" optionally seeds the group's per-level vocabulary as an array of {weight, name} entries — omit "id" and the server assigns a fresh ULID. Sorted by weight ascending at write time. Duplicate names return Conflict; whitespace-only names return BadRequest. The catalog is per-Group; weights are only comparable within one group. "rule" is a freeform instruction the AI must satisfy for every card created, decomposed, or moved into this group (e.g. "translate into Chinese; format as HTML"); omit for no rule.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in groupCreateIn) (*mcpsdk.CallToolResult, groupCreateOut, error) {
		g, err := m.backend.Group().Create(ctx, api.CreateGroupRequest{Name: in.Name, Rule: in.Rule, LevelCatalog: in.LevelCatalog})
		if err != nil {
			return nil, groupCreateOut{}, errors.Wrap(err)
		}
		return nil, groupCreateOut{Group: g}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "group.list",
		Description: "List all groups (flat). Each group includes its \"rule\" — the instruction the AI must satisfy for cards in that group.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, groupListOut, error) {
		gs, err := m.backend.Group().List(ctx)
		if err != nil {
			return nil, groupListOut{}, errors.Wrap(err)
		}
		return nil, groupListOut{Groups: gs}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "group.get",
		Description: `Get a single group by id, including its "rule" — the instruction every card created, decomposed, or moved into this group MUST satisfy (language, format, abstraction level). ALWAYS read the target group's rule before composing card content for it.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in groupGetIn) (*mcpsdk.CallToolResult, groupGetOut, error) {
		g, err := m.backend.Group().Get(ctx, in.ID)
		if err != nil {
			return nil, groupGetOut{}, errors.Wrap(err)
		}
		return nil, groupGetOut{Group: g}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "group.update",
		Description: `Update a group's name, rule, position, or level_catalog. "level_catalog" replaces the entire catalog (pass an empty array to clear it; omit to leave unchanged). This is the only path for humans to rename or delete catalog entries — AI tools should not use this to rename existing levels. "rule" replaces the group's freeform AI instruction (omit to leave unchanged).`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in groupUpdateIn) (*mcpsdk.CallToolResult, groupUpdateOut, error) {
		g, err := m.backend.Group().Update(ctx, in.ID, api.UpdateGroupRequest{
			Name:         in.Name,
			Rule:         in.Rule,
			Position:     in.Position,
			LevelCatalog: in.LevelCatalog,
		})
		if err != nil {
			return nil, groupUpdateOut{}, errors.Wrap(err)
		}
		return nil, groupUpdateOut{Group: g}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "group.delete",
		Description: "Delete a group. A group is non-empty when it holds any card, trashed ones included; pass recursive=true to delete those cards and the conversations anchored to them along with it.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in groupDeleteIn) (*mcpsdk.CallToolResult, groupDeleteOut, error) {
		if err := m.backend.Group().Delete(ctx, in.ID, in.Recursive); err != nil {
			return nil, groupDeleteOut{}, errors.Wrap(err)
		}
		return nil, groupDeleteOut{Deleted: true}, nil
	})
}

// ---- Tag ----

type tagListIn struct {
	GroupID string `json:"group_id"`
}
type tagListOut struct {
	Tags []*entity.Tag `json:"tags"`
}

type tagRenameIn struct {
	GroupID string `json:"group_id"`
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}
type tagRenameOut struct {
	Tag *entity.Tag `json:"tag"`
}

type tagDeleteIn struct {
	GroupID string `json:"group_id"`
	Name    string `json:"name"`
}
type tagDeleteOut struct {
	Deleted bool `json:"deleted"`
}

func (m *manager) registerTagTools() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "tag.list",
		Description: "List a group's tags. Tags are scoped to a group and lowercase-normalized; pass the group_id.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in tagListIn) (*mcpsdk.CallToolResult, tagListOut, error) {
		ts, err := m.backend.Tag().List(ctx, in.GroupID)
		if err != nil {
			return nil, tagListOut{}, errors.Wrap(err)
		}
		return nil, tagListOut{Tags: ts}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "tag.rename",
		Description: "Rename a tag within a group. If new_name already exists in that group, references merge into it.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in tagRenameIn) (*mcpsdk.CallToolResult, tagRenameOut, error) {
		t, err := m.backend.Tag().Rename(ctx, in.GroupID, in.OldName, api.RenameTagRequest{NewName: in.NewName})
		if err != nil {
			return nil, tagRenameOut{}, errors.Wrap(err)
		}
		return nil, tagRenameOut{Tag: t}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "tag.delete",
		Description: "Delete a tag from a group and detach it from that group's cards.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in tagDeleteIn) (*mcpsdk.CallToolResult, tagDeleteOut, error) {
		if err := m.backend.Tag().Delete(ctx, in.GroupID, in.Name); err != nil {
			return nil, tagDeleteOut{}, errors.Wrap(err)
		}
		return nil, tagDeleteOut{Deleted: true}, nil
	})
}

// ---- Card ----

type cardCreateIn struct {
	Title                string             `json:"title"`
	Summary              *string            `json:"summary,omitempty"`
	Content              string             `json:"content,omitempty"`
	Format               *string            `json:"format,omitempty"`
	LevelEntryID         *string            `json:"level_entry_id,omitempty"`
	Genesis              *string            `json:"genesis,omitempty"`
	GroupID              string             `json:"group_id"`
	Tags                 []string           `json:"tags,omitempty"`
	ParentCardID         *string            `json:"parent_card_id,omitempty"`
	SourceConversationID *string            `json:"source_conversation_id,omitempty"`
	Reference            *api.ReferenceSpec `json:"reference,omitempty"`
}
type cardCreateOut struct {
	Card      *entity.Card `json:"card"`
	GroupRule string       `json:"group_rule,omitempty"`
}

type cardListIn struct {
	GroupID        *string `json:"group_id,omitempty"`
	IncludeTrashed bool    `json:"include_trashed,omitempty"`
}
type cardListOut struct {
	Cards []*entity.Card `json:"cards"`
}

type cardChildrenIn struct {
	ID             string `json:"id"`
	IncludeTrashed bool   `json:"include_trashed,omitempty"`
}
type cardChildrenOut struct {
	Cards []*entity.Card `json:"cards"`
}

type cardGetIn struct {
	ID string `json:"id"`
}
type cardGetOut struct {
	Card *entity.Card `json:"card"`
}

type cardUpdateIn struct {
	ID              string    `json:"id"`
	Title           *string   `json:"title,omitempty"`
	Summary         *string   `json:"summary,omitempty"`
	Content         *string   `json:"content,omitempty"`
	Format          *string   `json:"format,omitempty"`
	LevelEntryID    *string   `json:"level_entry_id,omitempty"`
	ClearLevelEntry bool      `json:"clear_level_entry,omitempty"`
	Genesis         *string   `json:"genesis,omitempty"`
	GroupID         *string   `json:"group_id,omitempty"`
	Position        *int      `json:"position,omitempty"`
	Tags            *[]string `json:"tags,omitempty"`
}
type cardUpdateOut struct {
	Card      *entity.Card `json:"card"`
	GroupRule string       `json:"group_rule,omitempty"`
}

type cardDeleteIn struct {
	ID string `json:"id"`
}
type cardTrashIn struct {
	ID string `json:"id"`
	// Cascade defaults to true when omitted — matches the REST default and
	// the SPA behavior. Set false to trash only the target card and leave
	// its descendants live.
	Cascade *bool `json:"cascade,omitempty"`
}
type cardDeleteOut struct {
	Deleted bool `json:"deleted"`
}

type cardRestoreOut struct {
	Card *entity.Card `json:"card"`
}
type cardPurgeOut struct {
	Purged bool `json:"purged"`
}
type trashListIn struct {
	Limit int `json:"limit,omitempty"`
}
type trashListOut struct {
	Cards []*entity.Card `json:"cards"`
}
type trashEmptyIn struct{}
type trashEmptyOut struct {
	Purged int `json:"purged"`
}

type cardReorderIn struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}
type cardReorderOut struct {
	Card *entity.Card `json:"card"`
}

func (m *manager) registerCardTools() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card.create",
		Description: `Create a new card. Pass tags[] to attach existing or newly-created tags. For derived cards (spun off from a conversation), pass parent_card_id and source_conversation_id. When you create a card derived from a user's selection in another card, anchor it inline: set parent_card_id + source_conversation_id (optional) + reference: {selection_text: "<verbatim>"}. The backend creates the highlight in the same transaction; you do NOT need a follow-up reference.create. Use standalone reference.create only when adding a reference to a card that was created earlier (back-fill case). "format" is optional and may be "html", "markdown", or "text"; markdown renders with full styling (headings, lists, code, tables, blockquotes). Follow the target group's "rule" for the required format — default to "markdown" when the rule is silent, and use "text" only for literal preformatted content that must not be interpreted. "level_entry_id" attaches the card to an existing catalog entry in its group — call group.list to look up ids. To add a new catalog entry, call group.update first with the desired weight+name; the response includes the assigned id. "genesis" is a free-form human-readable note about where this card came from; defaults to "". **Genesis MUST NOT include raw card IDs, conversation IDs, or any ULID.** Show provenance via a title breadcrumb instead — e.g. "Decomposed from Zen roadmap - v0.12 planning - v0.12 spec" or "Extracted from user chat about SectionGradePill". Titles are what the reader sees on the tile; IDs are unreadable noise. "summary" is an optional short line (aim for <30 words; hard cap 500 chars) shown on the card tile in place of the auto content preview — omit to fall back to the preview. The target group may define a "rule" you MUST satisfy (language, format, abstraction level). Read it via group.get before composing; the result echoes "group_rule" when the destination group has one — if it is non-empty, verify the card conforms and card.update it if not.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in cardCreateIn) (*mcpsdk.CallToolResult, cardCreateOut, error) {
		c, err := m.backend.Card().Create(ctx, api.CreateCardRequest{
			Title: in.Title, Content: in.Content, Format: in.Format,
			LevelEntryID: in.LevelEntryID,
			Genesis:      in.Genesis,
			Summary:              in.Summary,
			GroupID:              in.GroupID,
			Tags:                 in.Tags,
			ParentCardID:         in.ParentCardID,
			SourceConversationID: in.SourceConversationID,
			Reference:            in.Reference,
		})
		if err != nil {
			return nil, cardCreateOut{}, errors.Wrap(err)
		}
		return nil, cardCreateOut{Card: c, GroupRule: m.groupRule(ctx, in.GroupID)}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card.list",
		Description: "List cards, optionally filtered by group_id. Pass include_trashed=true to include soft-deleted cards.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in cardListIn) (*mcpsdk.CallToolResult, cardListOut, error) {
		cs, err := m.backend.Card().List(ctx, in.GroupID, in.IncludeTrashed)
		if err != nil {
			return nil, cardListOut{}, errors.Wrap(err)
		}
		return nil, cardListOut{Cards: cs}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card.children",
		Description: `List the child cards of a container (a card previously decomposed), ordered by position ascending. Pass include_trashed=true to include soft-deleted children.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in cardChildrenIn) (*mcpsdk.CallToolResult, cardChildrenOut, error) {
		cs, err := m.backend.Card().Children(ctx, in.ID, in.IncludeTrashed)
		if err != nil {
			return nil, cardChildrenOut{}, errors.Wrap(err)
		}
		return nil, cardChildrenOut{Cards: cs}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card.get",
		Description: "Get a card by id, including its tags. Works on soft-deleted cards too.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in cardGetIn) (*mcpsdk.CallToolResult, cardGetOut, error) {
		c, err := m.backend.Card().Get(ctx, in.ID)
		if err != nil {
			return nil, cardGetOut{}, errors.Wrap(err)
		}
		return nil, cardGetOut{Card: c}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card.update",
		Description: `Update one or more fields of a card. Passing tags replaces the entire tag set on THIS card (omit to leave unchanged). **Tag additions cascade to every live descendant** — tagging a container also tags its sections and their children. Tag removals do NOT cascade — a container that drops a tag leaves descendants untouched. "format" may switch between "html", "markdown", and "text"; markdown renders with full styling. Follow the card's group "rule" for the required format. "level_entry_id" targets an existing catalog entry (call group.list to look up ids; use group.update to add a new one). Pass "clear_level_entry": true to detach the card from its catalog entry (Unfiled). "genesis" overrides the card's provenance note. **Genesis MUST NOT include raw card IDs, conversation IDs, or any ULID** — use a title breadcrumb like "Decomposed from Zen roadmap - v0.12 planning - v0.12 spec" instead. "summary" replaces the tile-preview line (empty string clears it, causing the auto content preview to show again). Never rename or delete existing catalog entries from card.update — use group.update for that. When you change "group_id" (move the card), the destination group's "rule" applies: the result echoes "group_rule" — if it is non-empty, transform the card to satisfy it (language, format, level) and card.update again.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in cardUpdateIn) (*mcpsdk.CallToolResult, cardUpdateOut, error) {
		c, err := m.backend.Card().Update(ctx, in.ID, api.UpdateCardRequest{
			Title: in.Title, Content: in.Content, Format: in.Format,
			Summary:         in.Summary,
			LevelEntryID:    in.LevelEntryID,
			ClearLevelEntry: in.ClearLevelEntry,
			Genesis:         in.Genesis,
			GroupID:         in.GroupID,
			Position:        in.Position,
			Tags:            in.Tags,
		})
		if err != nil {
			return nil, cardUpdateOut{}, errors.Wrap(err)
		}
		return nil, cardUpdateOut{Card: c, GroupRule: m.groupRule(ctx, c.GroupID)}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card.reorder",
		Description: `Move a section card to a new position inside its parent container. The parent does not change. Sibling positions shift atomically. Rejects cards whose parent_card_id is null (top-level cards use a different reorder surface). "position" is clamped to [0, siblingCount-1]; if the target equals the current position the call is a no-op.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in cardReorderIn) (*mcpsdk.CallToolResult, cardReorderOut, error) {
		c, err := m.backend.Card().Reorder(ctx, in.ID, in.Position)
		if err != nil {
			return nil, cardReorderOut{}, errors.Wrap(err)
		}
		return nil, cardReorderOut{Card: c}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card.delete",
		Description: `Soft-delete a card. The card is hidden from default lists + search but can be restored from the Trash. By default this cascades to every live descendant reached via parent_card_id (matching "move folder to trash"); pass "cascade": false to trash only the target card. Use card.purge to hard-delete after soft-deletion.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in cardTrashIn) (*mcpsdk.CallToolResult, cardDeleteOut, error) {
		cascade := true
		if in.Cascade != nil {
			cascade = *in.Cascade
		}
		if err := m.backend.Card().Delete(ctx, in.ID, cascade); err != nil {
			return nil, cardDeleteOut{}, errors.Wrap(err)
		}
		return nil, cardDeleteOut{Deleted: true}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card.restore",
		Description: `Restore a soft-deleted card (clears deleted_at). Errors if the card is live (not in trash).`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in cardDeleteIn) (*mcpsdk.CallToolResult, cardRestoreOut, error) {
		c, err := m.backend.Card().Restore(ctx, in.ID)
		if err != nil {
			return nil, cardRestoreOut{}, errors.Wrap(err)
		}
		return nil, cardRestoreOut{Card: c}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card.purge",
		Description: `Hard-delete a soft-deleted card. Errors if the card is live (call card.delete first).`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in cardDeleteIn) (*mcpsdk.CallToolResult, cardPurgeOut, error) {
		if err := m.backend.Card().Purge(ctx, in.ID); err != nil {
			return nil, cardPurgeOut{}, errors.Wrap(err)
		}
		return nil, cardPurgeOut{Purged: true}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "trash.list",
		Description: `List soft-deleted cards across all groups, ordered by deleted_at descending. Default limit 100.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in trashListIn) (*mcpsdk.CallToolResult, trashListOut, error) {
		limit := in.Limit
		if limit <= 0 {
			limit = 100
		}
		resp, err := m.backend.Card().Trash(ctx, limit)
		if err != nil {
			return nil, trashListOut{}, errors.Wrap(err)
		}
		return nil, trashListOut{Cards: resp.Cards}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "trash.empty",
		Description: `Hard-delete every soft-deleted card in one transaction (cascades to their tags + references). Returns the count of cards that were purged. Irreversible; use with intent.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ trashEmptyIn) (*mcpsdk.CallToolResult, trashEmptyOut, error) {
		resp, err := m.backend.Card().EmptyTrash(ctx)
		if err != nil {
			return nil, trashEmptyOut{}, errors.Wrap(err)
		}
		return nil, trashEmptyOut{Purged: resp.Purged}, nil
	})
}

// ---- Conversation ----

type conversationListIn struct {
	AnchorKind *string `json:"anchor_kind,omitempty"`
	AnchorID   *string `json:"anchor_id,omitempty"`
	Pending    bool    `json:"pending,omitempty"`
	Limit      int     `json:"limit,omitempty"`
}

type conversationListOut struct {
	Conversations    []*entity.Conversation `json:"conversations"`
	UnansweredCounts []int                  `json:"unanswered_counts,omitempty"`
}

type conversationGetIn struct {
	ID           string `json:"id"`
	MessageLimit int    `json:"message_limit,omitempty"`
}

type conversationGetOut struct {
	Conversation *entity.Conversation `json:"conversation"`
	Messages     []*entity.Message    `json:"messages"`
}


func (m *manager) registerConversationTools() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "conversation.list",
		Description: "List conversations. Pass pending=true to triage what's been waiting for you. Pass anchor_kind + anchor_id to see all conversations on a specific Card / Group. Pass nothing for all (most recent first).",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in conversationListIn) (*mcpsdk.CallToolResult, conversationListOut, error) {
		resp, err := m.backend.Conversation().List(ctx, in.AnchorKind, in.AnchorID, in.Pending, in.Limit)
		if err != nil {
			return nil, conversationListOut{}, errors.Wrap(err)
		}
		return nil, conversationListOut{
			Conversations:    resp.Conversations,
			UnansweredCounts: resp.UnansweredCounts,
		}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "conversation.get",
		Description: "Fetch a conversation with its messages. A channel event delivers only the user's current message and its ids — no history — so call this when you need the earlier thread. message_limit defaults to 100 if omitted.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in conversationGetIn) (*mcpsdk.CallToolResult, conversationGetOut, error) {
		conv, err := m.backend.Conversation().Get(ctx, in.ID)
		if err != nil {
			return nil, conversationGetOut{}, errors.Wrap(err)
		}
		limit := in.MessageLimit
		if limit <= 0 {
			limit = 100
		}
		msgs, err := m.backend.Conversation().GetMessages(ctx, in.ID, limit)
		if err != nil {
			return nil, conversationGetOut{}, errors.Wrap(err)
		}
		return nil, conversationGetOut{Conversation: conv, Messages: msgs.Messages}, nil
	})

}

// ---- Search ----

type searchIn struct {
	Query string `json:"query"`
	Scope string `json:"scope,omitempty"`
	Limit int    `json:"limit,omitempty"`
}
type searchOut struct {
	Query string              `json:"query"`
	Scope string              `json:"scope"`
	Cards []*entity.SearchHit `json:"cards"`
}

func (m *manager) registerSearchTools() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "search",
		Description: "Full-text search across cards. Returns snippets with <mark> markers around matches. Excludes soft-deleted cards.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in searchIn) (*mcpsdk.CallToolResult, searchOut, error) {
		resp, err := m.backend.Search().Search(ctx, in.Query, in.Scope, in.Limit)
		if err != nil {
			return nil, searchOut{}, errors.Wrap(err)
		}
		return nil, searchOut{
			Query: resp.Query, Scope: resp.Scope,
			Cards: resp.Cards,
		}, nil
	})
}

// ---- decompose ----

type decomposeIn struct {
	ParentCardID     string         `json:"parent_card_id"`
	ContainerContent *string        `json:"container_content,omitempty"`
	Cards            []api.CardSpec `json:"cards"`
}
type decomposeOut struct {
	Cards     []*entity.Card `json:"cards"`
	GroupRule string         `json:"group_rule,omitempty"`
}

func (m *manager) registerDecomposeTool() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "decompose",
		Description: `Transactionally split a card into N child cards. The parent stays live; by default its content is CLEARED (empty container) — opening the parent renders the ordered composition of its children. Optionally pass container_content to keep a verbatim leading metadata slice (the document's preamble — e.g. a date/status block) on the parent: it renders above the sections and is NOT a section (never graded, reordered, or counted). container_content is pure metadata (no logic or explanation) and is valid ONLY on a top-level document — a non-empty value when decomposing a nested card is rejected with BadRequest. Rejects re-decompose: a card that already has live children cannot be decomposed again. Each cards[] entry may set title, content, format, level_entry_id, tags, position, genesis, summary; format and group_id inherit from parent unless overridden. Format inherits from the parent unless a child overrides it; markdown renders with full styling, and the children's group "rule" governs the required format. Default position is the cards[] index (0-based). Default genesis is "Decomposed from <ancestor title chain>" — e.g. "Decomposed from Zen roadmap - v0.12 planning - v0.12 spec". Any genesis you override MUST also use titles, never IDs. Decompose is for **lossless splitting**, not synthesis: each child's body should be a verbatim slice of the parent's body. A child's title lives in the "title" field only — if its "content" begins with that same title as a heading, the leading heading is stripped so it is not duplicated in the body (idempotent when already absent). If any spec fails validation, ALL changes roll back. The children's group may define a "rule" you MUST satisfy (language, format, abstraction level). Read it via group.get before composing; the result echoes "group_rule" when that group has one — if it is non-empty, verify each child conforms and card.update it if not.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in decomposeIn) (*mcpsdk.CallToolResult, decomposeOut, error) {
		resp, err := m.backend.Card().Decompose(ctx, api.DecomposeRequest{
			ParentCardID:     in.ParentCardID,
			ContainerContent: in.ContainerContent,
			Cards:            in.Cards,
		})
		if err != nil {
			return nil, decomposeOut{}, errors.Wrap(err)
		}
		out := decomposeOut{Cards: resp.Cards}
		if len(resp.Cards) > 0 {
			out.GroupRule = m.groupRule(ctx, resp.Cards[0].GroupID)
		}
		return nil, out, nil
	})
}

// ---- compose ----

type composeIn struct {
	SourceCardIDs []string     `json:"source_card_ids"`
	Target        api.CardSpec `json:"target"`
}
type composeOut struct {
	Card *entity.Card `json:"card"`
}

func (m *manager) registerComposeTool() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "compose",
		Description: `Transactionally merge N source cards into 1 target card by **lossless joining**. The AI's only creative act is picking a good order for source_card_ids; target.content must be the source bodies concatenated verbatim in that order — NOT rewritten, NOT summarized, NOT paraphrased. Compose is the symmetric inverse of decompose: round-tripping decompose → compose should reproduce the original parent's body modulo per-card wrapping. For HTML sources, strip per-child <style> blocks and reuse one shared style block + wrapper class on the composed card. Sources must all live in the same group and must all be live (not soft-deleted); target inherits the sources' group_id unless overridden. The sources are soft-deleted (recoverable via Trash); target.genesis defaults to "Composed from <source1 title>, <source2 title>, ..." — titles, never IDs. Overriding genesis MUST follow the same rule. If any spec fails validation, ALL changes roll back.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in composeIn) (*mcpsdk.CallToolResult, composeOut, error) {
		resp, err := m.backend.Card().Compose(ctx, api.ComposeRequest{
			SourceCardIDs: in.SourceCardIDs,
			Target:        in.Target,
		})
		if err != nil {
			return nil, composeOut{}, errors.Wrap(err)
		}
		return nil, composeOut{Card: resp.Card}, nil
	})
}

// ---- reference ----

type referenceCreateIn struct {
	SourceCardID   string `json:"source_card_id"`
	DerivedCardID  string `json:"derived_card_id"`
	ConversationID string `json:"conversation_id"`
	SelectionText  string `json:"selection_text"`
}
type referenceOut struct {
	Reference *entity.Reference `json:"reference"`
}
type referenceListIn struct {
	SourceCardID   *string `json:"source_card_id,omitempty"`
	DerivedCardID  *string `json:"derived_card_id,omitempty"`
	ConversationID *string `json:"conversation_id,omitempty"`
}
type referenceListOut struct {
	References []*entity.Reference `json:"references"`
}
type referenceIDIn struct {
	ID string `json:"id"`
}
type referenceDeleteOut struct{}

func (m *manager) registerReferenceTools() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "reference.create",
		Description: `Anchor an AI-derived card to the verbatim selection the user highlighted in the source card. Call this AFTER card.create succeeds for the derivation. Pass the SAME selection_text the user provided — verbatim, not paraphrased. The reference produces a clickable highlight on the source card in the SPA that opens the conversation that produced the derivation.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in referenceCreateIn) (*mcpsdk.CallToolResult, referenceOut, error) {
		r, err := m.backend.Reference().Create(ctx, api.CreateReferenceRequest{
			SourceCardID:   in.SourceCardID,
			DerivedCardID:  in.DerivedCardID,
			ConversationID: in.ConversationID,
			SelectionText:  in.SelectionText,
		})
		if err != nil {
			return nil, referenceOut{}, errors.Wrap(err)
		}
		return nil, referenceOut{Reference: r}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "reference.get",
		Description: `Fetch a reference by id.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in referenceIDIn) (*mcpsdk.CallToolResult, referenceOut, error) {
		r, err := m.backend.Reference().Get(ctx, in.ID)
		if err != nil {
			return nil, referenceOut{}, errors.Wrap(err)
		}
		return nil, referenceOut{Reference: r}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "reference.list",
		Description: `List references filtered by any combination of source_card_id, derived_card_id, conversation_id. At least one filter is required.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in referenceListIn) (*mcpsdk.CallToolResult, referenceListOut, error) {
		refs, err := m.backend.Reference().List(ctx, api.ListReferencesRequest{
			SourceCardID:   in.SourceCardID,
			DerivedCardID:  in.DerivedCardID,
			ConversationID: in.ConversationID,
		})
		if err != nil {
			return nil, referenceListOut{}, errors.Wrap(err)
		}
		return nil, referenceListOut{References: refs}, nil
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "reference.delete",
		Description: `Delete a reference. Does not touch the source card, derived card, or conversation.`,
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in referenceIDIn) (*mcpsdk.CallToolResult, referenceDeleteOut, error) {
		if err := m.backend.Reference().Delete(ctx, in.ID); err != nil {
			return nil, referenceDeleteOut{}, errors.Wrap(err)
		}
		return nil, referenceDeleteOut{}, nil
	})
}
