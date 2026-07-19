package zenbackend

import (
	"context"
	"net/url"
	"strconv"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

type conversationClient struct{ c *client }

func (cc *conversationClient) Create(ctx context.Context, req api.CreateConversationRequest) (*entity.Conversation, error) {
	var out entity.Conversation
	if err := cc.c.doJSON(ctx, "POST", "/conversations", req, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (cc *conversationClient) Get(ctx context.Context, id string) (*entity.Conversation, error) {
	var out entity.Conversation
	if err := cc.c.doJSON(ctx, "GET", "/conversations/"+id, nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (cc *conversationClient) List(ctx context.Context, anchorKind, anchorID *string, pending bool, limit int) (*api.ConversationListResponse, error) {
	q := url.Values{}
	if anchorKind != nil {
		q.Set("anchor_kind", *anchorKind)
	}
	if anchorID != nil {
		q.Set("anchor_id", *anchorID)
	}
	if pending {
		q.Set("pending", "true")
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path := "/conversations"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out api.ConversationListResponse
	if err := cc.c.doJSON(ctx, "GET", path, nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (cc *conversationClient) UpdateTitle(ctx context.Context, id string, req api.UpdateConversationTitleRequest) (*entity.Conversation, error) {
	var out entity.Conversation
	if err := cc.c.doJSON(ctx, "PUT", "/conversations/"+id, req, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (cc *conversationClient) Delete(ctx context.Context, id string) error {
	if err := cc.c.doNoContent(ctx, "DELETE", "/conversations/"+id, nil); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func (cc *conversationClient) AppendMessage(ctx context.Context, conversationID string, req api.AppendMessageRequest) (*entity.Message, error) {
	var out entity.Message
	if err := cc.c.doJSON(ctx, "POST", "/conversations/"+conversationID+"/messages", req, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (cc *conversationClient) GetMessages(ctx context.Context, conversationID string, limit int) (*api.MessageListResponse, error) {
	path := "/conversations/" + conversationID + "/messages"
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}
	var out api.MessageListResponse
	if err := cc.c.doJSON(ctx, "GET", path, nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

