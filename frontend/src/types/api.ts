import type { Card, LevelEntry, SearchHit } from './entity';

export interface SearchResponse {
  query: string;
  scope: string;
  cards: SearchHit[];
  messages?: SearchHit[];
}

export interface BackendErrorBody {
  source?: string;
  status?: number;
  kind?: string;
  message?: string;
}

export class BackendError extends Error {
  constructor(
    public readonly status: number,
    public readonly kind: string,
    message: string,
  ) {
    super(message);
    this.name = 'BackendError';
  }
}

export interface CreateGroupRequest {
  name: string;
  rule?: string;
  level_catalog?: LevelEntry[];
}

export interface UpdateGroupRequest {
  name?: string;
  rule?: string;
  position?: number;
  level_catalog?: LevelEntry[];
}

export interface CreateCardRequest {
  title: string;
  content: string;
  format?: 'markdown' | 'html' | 'text';
  level_entry_id?: string | null;
  genesis?: string;
  group_id: string;
  tags?: string[];
  parent_card_id?: string;
  source_conversation_id?: string;
}

export interface UpdateCardRequest {
  title?: string;
  content?: string;
  format?: 'markdown' | 'html' | 'text';
  level_entry_id?: string | null;
  clear_level_entry?: boolean;
  genesis?: string;
  group_id?: string;
  position?: number;
  tags?: string[];
}

export interface ReorderCardRequest {
  position: number;
}

export interface ReviewCardRequest {
  grade: 'LGTM' | 'DIGESTED' | 'GRILLED';
}

export interface CardSpec {
  title: string;
  content?: string;
  format?: 'markdown' | 'html' | 'text';
  level_entry_id?: string | null;
  genesis?: string;
  group_id?: string;
  tags?: string[];
  position?: number;
}

export interface DecomposeRequest {
  cards: CardSpec[];
}

export interface DecomposeResponse {
  cards: Card[];
}

export interface TrashResponse {
  cards: Card[];
}

export interface RenameTagRequest {
  new_name: string;
}

export interface CreateConversationRequest {
  title: string;
  anchor_kind?: string | null;
  anchor_id?: string | null;
}

export interface UpdateConversationTitleRequest {
  title: string;
}

export interface AppendMessageRequest {
  role: 'user' | 'assistant' | 'system';
  content: string;
  selection_text?: string | null;
  // Addresses the message at one live Claude Code session. Omitted means the
  // message posts undelivered — it does not mean "broadcast".
  target_session_id?: string;
}

export interface DispatchRequest {
  target_session_id: string;
}

export interface ConversationListResponse {
  conversations: import('./entity').Conversation[];
  unanswered_counts?: number[];
}

export interface MessageListResponse {
  messages: import('./entity').Message[];
}
