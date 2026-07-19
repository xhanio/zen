import { describe, it, expect, vi, beforeEach } from 'vitest';
import { BackendError } from '../types/api';
import {
  listGroups,
  getGroup,
  search,
  listCards,
  createGroup,
  updateGroup,
  deleteGroup,
  createCard,
  updateCard,
  deleteCard,
  renameTag,
  deleteTag,
} from './client';
import type { Group } from '../types/entity';
import type { SearchResponse } from '../types/api';

const realFetch = global.fetch;

function mockJSON(status: number, body: unknown): typeof fetch {
  return vi.fn().mockResolvedValueOnce({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
    text: async () => JSON.stringify(body),
  } as Response) as unknown as typeof fetch;
}

beforeEach(() => {
  global.fetch = realFetch;
});

describe('listGroups', () => {
  it('parses a list of groups', async () => {
    const groups: Group[] = [
      { id: 'g1', name: 'work', rule: '', position: 0, created_at: 't', updated_at: 't', level_catalog: [] },
    ];
    global.fetch = mockJSON(200, groups);
    const got = await listGroups();
    expect(got).toEqual(groups);
  });
});

describe('getGroup', () => {
  it('hits /api/v1/groups/:id', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ id: 'g1', name: 'work', position: 0, created_at: 't', updated_at: 't', level_catalog: [] }),
      text: async () => '',
    } as Response);
    global.fetch = fetchMock as unknown as typeof fetch;
    await getGroup('g1');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/groups/g1', expect.any(Object));
  });

  it('throws BackendError with kind and status on 404', async () => {
    global.fetch = mockJSON(404, { source: 'zen-backend', status: 404, kind: 'NotFound', message: 'not here' });
    await expect(getGroup('missing')).rejects.toMatchObject({
      name: 'BackendError',
      status: 404,
      kind: 'NotFound',
      message: 'not here',
    });
  });
});

describe('listCards', () => {
  it('encodes group_id and include_trashed query params', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => [],
      text: async () => '[]',
    } as Response);
    global.fetch = fetchMock as unknown as typeof fetch;
    await listCards({ groupId: 'g1', includeTrashed: true });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/cards?group_id=g1&include_trashed=true',
      expect.any(Object),
    );
  });
});

describe('search', () => {
  it('parses a SearchResponse', async () => {
    const resp: SearchResponse = {
      query: 'hover', scope: 'all',
      cards: [{ kind: 'card', id: 'c1', title: 'x', snippet: 'y', group_id: 'g1' }],
    };
    global.fetch = mockJSON(200, resp);
    const got = await search('hover', 'all', 10);
    expect(got.cards).toHaveLength(1);
    expect(got.cards[0].snippet).toBe('y');
  });
});

describe('BackendError', () => {
  it('is an Error subclass', () => {
    const e = new BackendError(409, 'Conflict', 'taken');
    expect(e).toBeInstanceOf(Error);
    expect(e.status).toBe(409);
  });
});

function mockFetch(status: number, body: unknown): ReturnType<typeof vi.fn> {
  return vi.fn().mockResolvedValueOnce({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
    text: async () => JSON.stringify(body),
  } as Response);
}

function mock204(): ReturnType<typeof vi.fn> {
  return vi.fn().mockResolvedValueOnce({
    ok: true,
    status: 204,
    json: async () => null,
    text: async () => '',
  } as Response);
}

describe('group write methods', () => {
  it('createGroup POSTs JSON and returns the new group', async () => {
    const fetchMock = mockFetch(200, { id: 'g1', name: 'New', position: 0, created_at: 't', updated_at: 't', level_catalog: [] });
    global.fetch = fetchMock as unknown as typeof fetch;
    const result = await createGroup({ name: 'New' });
    expect(result.id).toBe('g1');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/groups', expect.objectContaining({ method: 'POST' }));
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(JSON.parse(init.body as string)).toEqual({ name: 'New' });
  });

  it('updateGroup PUTs to /groups/:id', async () => {
    const fetchMock = mockFetch(200, { id: 'g1', name: 'Renamed', position: 0, created_at: 't', updated_at: 't', level_catalog: [] });
    global.fetch = fetchMock as unknown as typeof fetch;
    const result = await updateGroup('g1', { name: 'Renamed' });
    expect(result.name).toBe('Renamed');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/groups/g1', expect.objectContaining({ method: 'PUT' }));
  });

  it('deleteGroup always passes recursive=true', async () => {
    const fetchMock = mock204();
    global.fetch = fetchMock as unknown as typeof fetch;
    await deleteGroup('g1');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/groups/g1?recursive=true', expect.objectContaining({ method: 'DELETE' }));
  });

  it('deleteGroup throws BackendError on 409', async () => {
    global.fetch = mockFetch(409, { status: 409, kind: 'Conflict', message: 'non-empty group' }) as unknown as typeof fetch;
    await expect(deleteGroup('g1')).rejects.toMatchObject({ status: 409, kind: 'Conflict' });
  });
});

describe('card write methods', () => {
  it('createCard POSTs JSON with tags', async () => {
    const fetchMock = mockFetch(200, {
      id: 'c1', title: 'T', content: 'C', group_id: 'g1',
      position: 0, tags: ['a'], created_at: 't', updated_at: 't',
    });
    global.fetch = fetchMock as unknown as typeof fetch;
    await createCard({ title: 'T', content: 'C', group_id: 'g1', tags: ['a'] });
    const body = JSON.parse((fetchMock.mock.calls[0][1] as RequestInit).body as string);
    expect(body.tags).toEqual(['a']);
  });

  it('updateCard accepts genesis', async () => {
    const fetchMock = mockFetch(200, {
      id: 'c1', title: 'T', content: 'C', group_id: 'g1',
      position: 0, tags: [], created_at: 't', updated_at: 't',
    });
    global.fetch = fetchMock as unknown as typeof fetch;
    await updateCard('c1', { genesis: 'note' });
    const body = JSON.parse((fetchMock.mock.calls[0][1] as RequestInit).body as string);
    expect(body.genesis).toBe('note');
  });

  it('deleteCard returns void on 204', async () => {
    global.fetch = mock204() as unknown as typeof fetch;
    await expect(deleteCard('c1')).resolves.toBeUndefined();
  });

  it('updateCard forwards tags array verbatim', async () => {
    const fetchMock = mockFetch(200, {
      id: 'c1', title: 'T', content: '', group_id: 'g1', document_id: null,
      position: 0, tags: ['x'], created_at: 't', updated_at: 't',
    });
    global.fetch = fetchMock as unknown as typeof fetch;
    await updateCard('c1', { tags: ['x'] });
    const body = JSON.parse((fetchMock.mock.calls[0][1] as RequestInit).body as string);
    expect(body.tags).toEqual(['x']);
  });
});

describe('tag write methods', () => {
  it('renameTag PUTs to /groups/:id/tags/:name', async () => {
    const fetchMock = mockFetch(200, { id: 't1', group_id: 'G1', name: 'fresh' });
    global.fetch = fetchMock as unknown as typeof fetch;
    await renameTag('G1', 'old', { new_name: 'fresh' });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/groups/G1/tags/old', expect.objectContaining({ method: 'PUT' }));
  });

  it('deleteTag DELETEs /groups/:id/tags/:name', async () => {
    const fetchMock = mock204();
    global.fetch = fetchMock as unknown as typeof fetch;
    await deleteTag('G1', 'old');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/groups/G1/tags/old', expect.objectContaining({ method: 'DELETE' }));
  });
});

import {
  listConversations,
  createConversation,
  appendMessage,
} from './client';
import type { ConversationListResponse } from '../types/api';
import type { Conversation, Message } from '../types/entity';

describe('listConversations', () => {
  it('encodes pending filter', async () => {
    const fakeResp: ConversationListResponse = { conversations: [], unanswered_counts: [] };
    global.fetch = mockJSON(200, fakeResp);
    await listConversations({ pending: true });
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/conversations?pending=true'),
      expect.any(Object),
    );
  });
});

describe('createConversation', () => {
  it('roundtrips title + anchor', async () => {
    const conv: Conversation = {
      id: '01ABC', title: 'hi',
      anchor_kind: 'card', anchor_id: '01CARD',
      created_at: '2026-06-27T00:00:00Z',
      last_message_at: '2026-06-27T00:00:00Z',
    };
    global.fetch = mockJSON(201, conv);
    const got = await createConversation({ title: 'hi', anchor_kind: 'card', anchor_id: '01CARD' });
    expect(got.id).toBe('01ABC');
    expect(got.anchor_kind).toBe('card');
  });
});

describe('appendMessage', () => {
  it('posts to the conversation messages route', async () => {
    const msg: Message = {
      id: '01MSG', conversation_id: '01CONV',
      role: 'user', content: 'hi', selection_text: null,
      created_at: '2026-06-27T00:00:00Z',
    };
    global.fetch = mockJSON(201, msg);
    const got = await appendMessage('01CONV', { role: 'user', content: 'hi' });
    expect(got.id).toBe('01MSG');
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/conversations/01CONV/messages'),
      expect.objectContaining({ method: 'POST' }),
    );
  });
});
