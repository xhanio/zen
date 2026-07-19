import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { mount, flushPromises } from '@vue/test-utils';
import ContainerBody from './ContainerBody.vue';
import { useCardsStore } from '../stores/cards';
import { useTilePrefsStore } from '../stores/tilePrefs';
import { useContainerFilterStore } from '../stores/containerFilter';
import type { Card } from '../types/entity';

vi.mock('../api/client', () => ({
  listCards: vi.fn(),
  getCard: vi.fn(),
  createCard: vi.fn(),
  updateCard: vi.fn(),
  deleteCard: vi.fn(),
  restoreCard: vi.fn(),
  purgeCard: vi.fn(),
  listChildren: vi.fn(),
}));
import { listChildren } from '../api/client';

function stub(over: Partial<Card> = {}): Card {
  return {
    id: 'c', title: 'T', content: 'body', summary: '', format: 'markdown', level: null,
    genesis: '', deleted_at: null, group_id: 'g1', position: 0, tags: [],
    parent_card_id: null, source_conversation_id: null, created_at: '', updated_at: '',
    ...over,
  } as Card;
}

const routerLinkStub = { name: 'RouterLink', template: '<a><slot/></a>', props: ['to'] };

beforeEach(() => {
  localStorage.clear();
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('ContainerBody', () => {
  it('renders each live child via CardBody in position order and skips trashed by default', async () => {
    const parent = stub({ id: 'p', content: '' });
    (listChildren as any).mockResolvedValue([
      stub({ id: 'b', parent_card_id: 'p', position: 1, content: 'b-body' }),
      stub({ id: 'a', parent_card_id: 'p', position: 0, content: 'a-body' }),
      stub({ id: 'c', parent_card_id: 'p', position: 2, deleted_at: '2026-07-06T00:00:00Z', content: 'c-body' }),
    ]);
    const w = mount(ContainerBody, {
      props: { parent },
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          CardBody: { template: '<div class="card-body" :data-card-id="card.id"></div>', props: ['card'] },
        },
      },
    });
    await flushPromises();
    const rendered = w.findAll('.card-body').map((n) => n.attributes('data-card-id'));
    expect(rendered).toEqual(['a', 'b']);
  });

  it('a folded section renders title-only (no body, no grade shell)', async () => {
    const parent = stub({ id: 'p', content: '' });
    (listChildren as any).mockResolvedValue([
      stub({ id: 'a', parent_card_id: 'p', position: 0, content: 'a-body' }),
      stub({ id: 'b', title: 'Folded Section', parent_card_id: 'p', position: 1, content: 'b-body' }),
    ]);
    useContainerFilterStore().toggleCard('b');
    const w = mount(ContainerBody, {
      props: { parent },
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          CardBody: { template: '<div class="card-body" :data-card-id="card.id"></div>', props: ['card'] },
        },
      },
    });
    await flushPromises();
    // Expanded section a: body rendered + full grade shell.
    expect(w.findAll('.card-body').map((n) => n.attributes('data-card-id'))).toEqual(['a']);
    expect(w.findAll('[data-test="section-shell"]').length).toBe(1);
    // Folded section b: title still shown, marked collapsed, no body.
    const collapsed = w.find('[data-test="section-collapsed"]');
    expect(collapsed.exists()).toBe(true);
    expect(collapsed.text()).toContain('Folded Section');
  });

  it('clicking a section title toggles its collapse (per-section accordion)', async () => {
    const parent = stub({ id: 'p', content: '' });
    (listChildren as any).mockResolvedValue([
      stub({ id: 'a', title: 'Section A', parent_card_id: 'p', position: 0, content: 'a-body' }),
    ]);
    const cf = useContainerFilterStore();
    const w = mount(ContainerBody, {
      props: { parent },
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          CardBody: { template: '<div class="card-body"></div>', props: ['card'] },
        },
      },
    });
    await flushPromises();
    expect(cf.isCollapsed('a')).toBe(false);
    expect(w.find('.card-body').exists()).toBe(true);
    await w.find('[data-test="section-title"]').trigger('click');
    expect(cf.isCollapsed('a')).toBe(true);
    expect(w.find('.card-body').exists()).toBe(false); // body hidden when folded
  });

  it('renders an html section title as an <h2> in the card\'s own wrapper + style', async () => {
    const parent = stub({ id: 'p', content: '' });
    (listChildren as any).mockResolvedValue([
      stub({
        id: 'a', title: '变更', parent_card_id: 'p', position: 0, format: 'html',
        content: '<article class="zen-spec"><style>.zen-spec h2{color:red}</style><p>body</p></article>',
      }),
    ]);
    const w = mount(ContainerBody, {
      props: { parent },
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          CardBody: { template: '<div class="card-body"></div>', props: ['card'] },
          HtmlBody: { template: '<div class="title-html" :data-src="source"></div>', props: ['source'] },
        },
      },
    });
    await flushPromises();
    const src = w.find('.title-html').attributes('data-src') ?? '';
    expect(src).toContain('class="zen-spec"'); // the card's own wrapper
    expect(src).toContain('<style>');           // its own style block
    expect(src).toContain('<h2>变更</h2>');      // title as an h2
    expect(src).not.toContain('<p>body</p>');    // body excluded from the title render
  });

  it('shows a stub for each trashed child when showTrashedSections is on', async () => {
    const prefs = useTilePrefsStore();
    prefs.showTrashedSections = true;
    const parent = stub({ id: 'p', content: '' });
    (listChildren as any).mockResolvedValue([
      stub({ id: 'a', parent_card_id: 'p', position: 0, content: 'a' }),
      stub({ id: 'x', title: 'Removed', parent_card_id: 'p', position: 1, deleted_at: '2026-07-06T00:00:00Z' }),
    ]);
    const w = mount(ContainerBody, {
      props: { parent },
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          CardBody: { template: '<div class="card-body"></div>', props: ['card'] },
        },
      },
    });
    await flushPromises();
    expect(w.text()).toContain('Removed');
    expect(w.text()).toContain('Trash');
  });

  it('recurses when a child is itself a container', async () => {
    const parent = stub({ id: 'p', content: '' });
    const child = stub({ id: 'k', parent_card_id: 'p', content: '' });
    const grand = stub({ id: 'g', parent_card_id: 'k', content: 'grand body' });
    (listChildren as any).mockImplementation(async (id: string) => {
      if (id === 'p') return [child];
      if (id === 'k') return [grand];
      return [];
    });
    const store = useCardsStore();
    store.byID['p'] = parent;
    store.byID['k'] = child;
    store.byID['g'] = grand;
    const w = mount(ContainerBody, {
      props: { parent },
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          ContentBody: { template: '<div class="content-body">{{ source }}</div>', props: ['source', 'format', 'highlights'] },
        },
      },
    });
    await flushPromises();
    await flushPromises(); // let the nested ContainerBody mount + fetch
    // The recursive CardBody → ContainerBody → CardBody chain reaches the grandchild ContentBody:
    expect(w.text()).toContain('grand body');
  });
});

describe('ContainerBody reorder gesture', () => {
  it('ribbon drag on section 2 → drop on top-half of section 0 → reorderChild(id, 0)', async () => {
    const parent = stub({ id: 'p', content: '' });
    (listChildren as any).mockResolvedValue([
      stub({ id: 'a', parent_card_id: 'p', position: 0, content: 'a' }),
      stub({ id: 'b', parent_card_id: 'p', position: 1, content: 'b' }),
      stub({ id: 'c', parent_card_id: 'p', position: 2, content: 'c' }),
    ]);
    const store = useCardsStore();
    store.byID['a'] = stub({ id: 'a', parent_card_id: 'p', position: 0 });
    store.byID['b'] = stub({ id: 'b', parent_card_id: 'p', position: 1 });
    store.byID['c'] = stub({ id: 'c', parent_card_id: 'p', position: 2 });
    const spy = vi.spyOn(store, 'reorderChild').mockResolvedValue(store.byID['c']);
    const w = mount(ContainerBody, {
      props: { parent },
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          CardBody: { template: '<div class="card-body"></div>', props: ['card'] },
        },
      },
    });
    await flushPromises();

    const ribbons = w.findAll('[data-test="section-ribbon"]');
    expect(ribbons.length).toBe(3);
    const sections = w.findAll('[data-test="section-shell"]');
    expect(sections.length).toBe(3);

    // Fake the target section's bounding rect so the drop lands in the
    // top half (insert BEFORE section 0 → target position 0).
    const targetEl = sections[0].element as HTMLElement;
    targetEl.getBoundingClientRect = () =>
      ({ top: 100, left: 0, right: 0, bottom: 200, width: 0, height: 100, x: 0, y: 100, toJSON() {} } as DOMRect);

    const dt = {
      data: new Map<string, string>(),
      setData(t: string, v: string) { this.data.set(t, v); },
      getData(t: string) { return this.data.get(t) ?? ''; },
      effectAllowed: '', dropEffect: '',
    };
    const startEv = new Event('dragstart', { bubbles: true }) as unknown as DragEvent;
    Object.defineProperty(startEv, 'dataTransfer', { value: dt });
    ribbons[2].element.dispatchEvent(startEv);

    const overEv = new Event('dragover', { bubbles: true, cancelable: true }) as unknown as DragEvent;
    Object.defineProperty(overEv, 'dataTransfer', { value: dt });
    Object.defineProperty(overEv, 'preventDefault', { value: () => {} });
    Object.defineProperty(overEv, 'clientY', { value: 120 }); // top half of the 100..200 rect
    targetEl.dispatchEvent(overEv);

    const dropEv = new Event('drop', { bubbles: true, cancelable: true }) as unknown as DragEvent;
    Object.defineProperty(dropEv, 'dataTransfer', { value: dt });
    Object.defineProperty(dropEv, 'preventDefault', { value: () => {} });
    Object.defineProperty(dropEv, 'clientY', { value: 120 });
    targetEl.dispatchEvent(dropEv);
    await flushPromises();

    expect(spy).toHaveBeenCalledWith('c', 0);
  });
  // Regression: the gutter renders outside ContainerBody and depends on this
  // event to know where each section is. An earlier version cleared a
  // ref-populated array inside the emit's own watcher, so the anchors always
  // went out empty and every dot stacked at the top. Read the DOM instead.
  it('emits the live section elements to @anchors, in order', async () => {
    const parent = stub({ id: 'p', content: '' });
    (listChildren as any).mockResolvedValue([
      stub({ id: 'a', parent_card_id: 'p', position: 0, content: 'a-body' }),
      stub({ id: 'b', parent_card_id: 'p', position: 1, content: 'b-body' }),
    ]);
    const w = mount(ContainerBody, {
      props: { parent },
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          CardBody: { template: '<div class="card-body"></div>', props: ['card'] },
        },
      },
    });
    await flushPromises();
    const events = w.emitted('anchors') as Array<[HTMLElement[]]> | undefined;
    expect(events, 'no anchors event emitted').toBeTruthy();
    const last: HTMLElement[] = events![events!.length - 1][0];
    expect(last.length).toBe(2);
    expect(last.map((el) => el.getAttribute('data-card-id'))).toEqual(['a', 'b']);
  });

  it('escalates a section to DIGESTED on a plain click (no collapse dance)', async () => {
    const parent = stub({ id: 'p', content: '' });
    (listChildren as any).mockResolvedValue([stub({ id: 's', parent_card_id: 'p', position: 0, content: 'body' })]);
    const cards = useCardsStore();
    const esc = vi.spyOn(cards, 'escalateReviewGrade').mockResolvedValue();
    const w = mount(ContainerBody, {
      props: { parent },
      global: { stubs: { RouterLink: routerLinkStub, CardBody: { template: '<div class="card-body"/>', props: ['card'] } } },
    });
    await flushPromises();
    await w.find('[data-test="section-title"]').trigger('click');
    expect(esc).toHaveBeenCalledWith('s', 'DIGESTED');
  });

  it('escalates when the section body is clicked, not only the title', async () => {
    const parent = stub({ id: 'p', content: '' });
    (listChildren as any).mockResolvedValue([stub({ id: 's', parent_card_id: 'p', position: 0, content: 'body' })]);
    const cards = useCardsStore();
    const esc = vi.spyOn(cards, 'escalateReviewGrade').mockResolvedValue();
    const w = mount(ContainerBody, {
      props: { parent },
      global: { stubs: { RouterLink: routerLinkStub, CardBody: { template: '<div class="card-body"/>', props: ['card'] } } },
    });
    await flushPromises();
    await w.find('.card-body').trigger('click');
    expect(esc).toHaveBeenCalledWith('s', 'DIGESTED');
  });

});
