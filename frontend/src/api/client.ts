import type { Group, Tag, Card, SearchHit, Conversation, Message, ConversationEvent } from '../types/entity';
import {
  BackendError,
  type BackendErrorBody,
  type SearchResponse,
  type CreateGroupRequest,
  type UpdateGroupRequest,
  type CreateCardRequest,
  type UpdateCardRequest,
  type ReorderCardRequest,
  type ReviewCardRequest,
  type DecomposeRequest,
  type DecomposeResponse,
  type TrashResponse,
  type RenameTagRequest,
  type CreateConversationRequest,
  type UpdateConversationTitleRequest,
  type AppendMessageRequest,
  type DispatchRequest,
  type ConversationListResponse,
  type MessageListResponse,
} from '../types/api';

const BASE = '/api/v1';

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const init: RequestInit = { method, headers: { Accept: 'application/json' } };
  if (body !== undefined) {
    (init.headers as Record<string, string>)['Content-Type'] = 'application/json';
    init.body = JSON.stringify(body);
  }
  const resp = await fetch(BASE + path, init);
  if (!resp.ok) {
    let parsed: BackendErrorBody = {};
    try {
      parsed = await resp.json();
    } catch {
      /* non-JSON body — fall through with empty parsed */
    }
    throw new BackendError(
      parsed.status ?? resp.status,
      parsed.kind ?? '',
      parsed.message ?? `http ${resp.status}`,
    );
  }
  if (resp.status === 204) return undefined as T;
  return (await resp.json()) as T;
}

// ---- Group ----

export function listGroups(): Promise<Group[]> {
  return request('GET', '/groups');
}

export function getGroup(id: string): Promise<Group> {
  return request('GET', `/groups/${encodeURIComponent(id)}`);
}

// ---- Tag ----

export function listTags(groupId: string): Promise<Tag[]> {
  return request('GET', `/groups/${encodeURIComponent(groupId)}/tags`);
}

// ---- Card ----

export interface ListCardsOptions {
  groupId?: string;
  includeTrashed?: boolean;
}

export function listCards(opts: ListCardsOptions = {}): Promise<Card[]> {
  const params = new URLSearchParams();
  if (opts.groupId) params.set('group_id', opts.groupId);
  if (opts.includeTrashed) params.set('include_trashed', 'true');
  const qs = params.toString();
  return request('GET', `/cards${qs ? '?' + qs : ''}`);
}

export function getCard(id: string): Promise<Card> {
  return request('GET', `/cards/${encodeURIComponent(id)}`);
}

// ---- Search ----

export function search(query: string, scope?: string, limit?: number): Promise<SearchResponse> {
  const params = new URLSearchParams();
  params.set('q', query);
  if (scope) params.set('scope', scope);
  if (limit && limit > 0) params.set('limit', String(limit));
  return request('GET', `/search?${params.toString()}`);
}

// ---- Group writes ----

export function createGroup(req: CreateGroupRequest): Promise<Group> {
  return request('POST', '/groups', req);
}

export function updateGroup(id: string, req: UpdateGroupRequest): Promise<Group> {
  return request('PUT', `/groups/${encodeURIComponent(id)}`, req);
}

export function deleteGroup(id: string): Promise<void> {
  return request('DELETE', `/groups/${encodeURIComponent(id)}?recursive=true`);
}

// ---- Card writes ----

export function createCard(req: CreateCardRequest): Promise<Card> {
  return request('POST', '/cards', req);
}

export function updateCard(id: string, req: UpdateCardRequest): Promise<Card> {
  return request('PUT', `/cards/${encodeURIComponent(id)}`, req);
}

export function reorderCard(id: string, req: ReorderCardRequest): Promise<Card> {
  return request('POST', `/cards/${encodeURIComponent(id)}/reorder`, req);
}

export function reviewCard(id: string, req: ReviewCardRequest): Promise<Card> {
  return request('POST', `/cards/${encodeURIComponent(id)}/review`, req);
}

// deleteCard performs a SOFT delete (sets deleted_at; recoverable via Trash).
// cascade=true (default) also trashes every descendant reached via
// parent_card_id; cascade=false trashes just this card.
export function deleteCard(id: string, cascade = true): Promise<void> {
  const q = cascade ? '' : '?cascade=false';
  return request('DELETE', `/cards/${encodeURIComponent(id)}${q}`);
}

export function decomposeCard(parentCardID: string, req: DecomposeRequest): Promise<DecomposeResponse> {
  return request('POST', `/cards/${encodeURIComponent(parentCardID)}/decompose`, req);
}

export function restoreCard(id: string): Promise<Card> {
  return request('POST', `/cards/${encodeURIComponent(id)}/restore`);
}

export function purgeCard(id: string): Promise<void> {
  return request('DELETE', `/cards/${encodeURIComponent(id)}/purge`);
}

export function listTrash(limit?: number): Promise<TrashResponse> {
  const q = limit ? `?limit=${limit}` : '';
  return request('GET', `/trash${q}`);
}

export interface ChildrenResponse {
  cards: Card[];
}

export function listChildren(id: string, includeTrashed = false): Promise<Card[]> {
  const q = includeTrashed ? '?include_trashed=true' : '';
  return request<ChildrenResponse>('GET', `/cards/${encodeURIComponent(id)}/children${q}`).then((r) => r.cards);
}

export function emptyTrash(): Promise<{ purged: number }> {
  return request('DELETE', '/trash');
}

// ---- Tag writes ----

export function renameTag(groupId: string, oldName: string, req: RenameTagRequest): Promise<Tag> {
  return request('PUT', `/groups/${encodeURIComponent(groupId)}/tags/${encodeURIComponent(oldName)}`, req);
}

export function deleteTag(groupId: string, name: string): Promise<void> {
  return request('DELETE', `/groups/${encodeURIComponent(groupId)}/tags/${encodeURIComponent(name)}`);
}

// ---- Conversation ----

export interface ListConversationsOptions {
  anchorKind?: string;
  anchorID?: string;
  pending?: boolean;
  limit?: number;
}

export function listConversations(opts: ListConversationsOptions = {}): Promise<ConversationListResponse> {
  const params = new URLSearchParams();
  if (opts.anchorKind) params.set('anchor_kind', opts.anchorKind);
  if (opts.anchorID) params.set('anchor_id', opts.anchorID);
  if (opts.pending) params.set('pending', 'true');
  if (opts.limit && opts.limit > 0) params.set('limit', String(opts.limit));
  const qs = params.toString();
  return request('GET', `/conversations${qs ? '?' + qs : ''}`);
}

export function getConversation(id: string): Promise<Conversation> {
  return request('GET', `/conversations/${encodeURIComponent(id)}`);
}

export function createConversation(req: CreateConversationRequest): Promise<Conversation> {
  return request('POST', '/conversations', req);
}

export function updateConversationTitle(id: string, req: UpdateConversationTitleRequest): Promise<Conversation> {
  return request('PUT', `/conversations/${encodeURIComponent(id)}`, req);
}

export function deleteConversation(id: string): Promise<void> {
  return request('DELETE', `/conversations/${encodeURIComponent(id)}`);
}

export function listMessages(
  conversationID: string,
  opts: { after?: string; limit?: number } = {},
): Promise<MessageListResponse> {
  const params = new URLSearchParams();
  if (opts.after) params.set('after', opts.after);
  if (opts.limit && opts.limit > 0) params.set('limit', String(opts.limit));
  const q = params.toString() ? `?${params.toString()}` : '';
  return request('GET', `/conversations/${encodeURIComponent(conversationID)}/messages${q}`);
}

export function appendMessage(conversationID: string, req: AppendMessageRequest): Promise<Message> {
  return request('POST', `/conversations/${encodeURIComponent(conversationID)}/messages`, req);
}

// Re-publishes an already-stored message at a live session. Serves both the
// "nothing was connected when I posted" path and "resend to another session";
// neither duplicates the message.
export function dispatchMessage(
  conversationID: string,
  messageID: string,
  req: DispatchRequest,
): Promise<void> {
  return request(
    'POST',
    `/conversations/${encodeURIComponent(conversationID)}/messages/${encodeURIComponent(messageID)}/dispatch`,
    req,
  );
}

export type {
  Group, Tag, Card, SearchHit, SearchResponse,
  Conversation, Message, ConversationEvent,
};
export { BackendError };
