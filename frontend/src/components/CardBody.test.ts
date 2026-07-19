import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { mount } from '@vue/test-utils';
import CardBody from './CardBody.vue';
import { useCardsStore } from '../stores/cards';
import type { Card } from '../types/entity';

vi.mock('../api/client', () => ({
  listCards: vi.fn(),
  getCard: vi.fn(),
  createCard: vi.fn(),
  updateCard: vi.fn(),
  deleteCard: vi.fn(),
  restoreCard: vi.fn(),
  purgeCard: vi.fn(),
  listChildren: vi.fn().mockResolvedValue([]),
}));

function stub(over: Partial<Card> = {}): Card {
  return {
    id: 'c1', title: 'T', content: '', summary: '', format: 'markdown', level: null,
    genesis: '', deleted_at: null, group_id: 'g1', position: 0, tags: [],
    parent_card_id: null, source_conversation_id: null, created_at: '', updated_at: '',
    ...over,
  } as Card;
}

beforeEach(() => setActivePinia(createPinia()));

describe('CardBody dispatch', () => {
  it('renders ContainerBody when container', () => {
    const s = useCardsStore();
    const parent = stub({ id: 'p', content: '' });
    s.byChildren['p'] = [stub({ id: 'k1', parent_card_id: 'p', content: 'body' })];
    s.byID['p'] = parent;
    s.byID['k1'] = s.byChildren['p'][0];
    const w = mount(CardBody, {
      props: { card: parent },
      global: { stubs: { ContainerBody: { template: '<div data-test="cb-container"></div>' }, ContentBody: { template: '<div data-test="cb-content"></div>' } } },
    });
    expect(w.find('[data-test="cb-container"]').exists()).toBe(true);
    expect(w.find('[data-test="cb-content"]').exists()).toBe(false);
  });

  it('renders ContentBody when leaf', () => {
    const s = useCardsStore();
    const leaf = stub({ id: 'l', content: 'body' });
    s.byID['l'] = leaf;
    const w = mount(CardBody, {
      props: { card: leaf },
      global: { stubs: { ContainerBody: { template: '<div data-test="cb-container"></div>' }, ContentBody: { template: '<div data-test="cb-content"></div>' } } },
    });
    expect(w.find('[data-test="cb-content"]').exists()).toBe(true);
    expect(w.find('[data-test="cb-container"]').exists()).toBe(false);
  });

  it('renders ContainerBody for a container that has content (preamble)', () => {
    const s = useCardsStore();
    const parent = stub({ id: 'pp', content: 'Date: 2026-07-11' });
    s.byChildren['pp'] = [stub({ id: 'kk', parent_card_id: 'pp', content: 'body' })];
    s.byID['pp'] = parent;
    s.byID['kk'] = s.byChildren['pp'][0];
    const w = mount(CardBody, {
      props: { card: parent },
      global: { stubs: { ContainerBody: { template: '<div data-test="cb-container"></div>' }, ContentBody: { template: '<div data-test="cb-content"></div>' } } },
    });
    expect(w.find('[data-test="cb-container"]').exists()).toBe(true);
  });

  it('shows (empty) placeholder when content empty AND no children', () => {
    const s = useCardsStore();
    const empty = stub({ id: 'e', content: '' });
    s.byID['e'] = empty;
    const w = mount(CardBody, {
      props: { card: empty },
      global: { stubs: { ContainerBody: { template: '<div data-test="cb-container"></div>' }, ContentBody: { template: '<div data-test="cb-content"></div>' } } },
    });
    expect(w.text()).toContain('(empty)');
  });
});
