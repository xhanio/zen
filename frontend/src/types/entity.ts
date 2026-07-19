export interface LevelEntry {
  id: string;
  weight: number;
  name: string;
}

export interface Group {
  id: string;
  name: string;
  rule: string;
  position: number;
  level_catalog: LevelEntry[];
  created_at: string;
  updated_at: string;
}

export interface Tag {
  id: string;
  group_id: string;
  name: string;
  card_count: number;
}

export type EntityFormat = 'markdown' | 'html' | 'text';

export type ReviewGrade = 'LGTM' | 'DIGESTED' | 'GRILLED';

export interface Card {
  id: string;
  title: string;
  summary: string;
  content: string;
  format: EntityFormat;
  level_entry_id: string | null;
  genesis: string;
  deleted_at: string | null;
  group_id: string;
  position: number;
  tags: string[];
  parent_card_id: string | null;
  source_conversation_id: string | null;
  created_at: string;
  updated_at: string;
  review_grade: ReviewGrade;
  review_score: number | null;
  reviewed_at: string | null;
  references?: Reference[];
}

export interface Reference {
  id: string;
  source_card_id: string;
  derived_card_id: string;
  conversation_id: string | null;
  selection_text: string;
  created_at: string;
}

export interface SearchHit {
  kind: 'card' | 'message';
  id: string;
  title: string;
  // Ancestor card titles, root-first, excluding this card. Rendered as a
  // breadcrumb ahead of the title. Absent for top-level cards and messages.
  title_path?: string[];
  snippet: string;
  group_id: string;
  conversation_id?: string;
}

export interface Conversation {
  id: string;
  title: string;
  anchor_kind: string | null;
  anchor_id: string | null;
  created_at: string;
  last_message_at: string;
}

export interface Message {
  id: string;
  conversation_id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  selection_text: string | null;
  session_id?: string | null;
  session_cwd?: string | null;
  created_at: string;
}

export interface ConversationEvent {
  conversation_id: string;
  message_id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  selection_text?: string;
  session_id?: string;
  session_cwd?: string;
  // The message immediately before this one in the same conversation. Empty
  // means "no gap possible": the first message, and every dispatched re-publish.
  prev_message_id?: string;
  created_at: string;
}

// One live zen-mcp channel, as the backend's presence registry sees it.
// session_id is the routing identity and survives a channel restart;
// instance_id is per-process and does not.
export interface ChannelSession {
  instance_id: string;
  session_id: string;
  cwd: string;
  started_at: string;
  client_name: string;
  client_version: string;
  connected_at: string;
}
