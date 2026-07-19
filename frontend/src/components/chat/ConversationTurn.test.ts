import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import ConversationTurn from './ConversationTurn.vue';
import type { Message } from '../../types/entity';

const msg = (over: Partial<Message> = {}): Message => ({
  id: 'm1', conversation_id: 'c1', role: 'user', content: 'Why FTS5?',
  selection_text: null, created_at: '2026-07-10T14:14:00Z', ...over,
} as Message);

type TurnProps = {
  message: Message;
  speaker: string;
  state: 'sent' | 'delivered' | 'undelivered' | null;
  sessionTag?: string | null;
  sessionColor?: string | null;
};
const mountTurn = (props: TurnProps) =>
  mount(ConversationTurn, {
    props,
    global: { stubs: { MarkdownBody: { template: '<div class="md-stub"></div>', props: ['source'] } } },
  });

describe('ConversationTurn', () => {
  it('renders the accent ribbon for a user turn', () => {
    const w = mountTurn({ message: msg(), speaker: 'You', state: 'sent' });
    expect(w.find('[data-test="turn-ribbon"]').classes().join(' ')).toContain('bg-accent-fg');
  });

  it('shows the session tag on a user turn and a colour dot', () => {
    const w = mountTurn({
      message: msg(), speaker: 'You', state: 'sent',
      sessionTag: 'zen', sessionColor: 'hsl(200 65% 50%)',
    });
    expect(w.find('[data-test="turn-session"]').text()).toContain('zen');
    expect(w.find('[data-test="turn-session-dot"]').exists()).toBe(true);
  });

  it('shows no session tag when none is given', () => {
    const w = mountTurn({ message: msg(), speaker: 'You', state: 'sent' });
    expect(w.find('[data-test="turn-session"]').exists()).toBe(false);
    expect(w.find('[data-test="turn-session-dot"]').exists()).toBe(false);
  });

  it('renders Claude bodies through MarkdownBody and yours as plain text', () => {
    const ai = mountTurn({ message: msg({ role: 'assistant', content: '# Heading' }), speaker: 'sonnet-4', state: null });
    expect(ai.find('.md-stub').exists()).toBe(true);
    const you = mountTurn({ message: msg({ content: 'plain' }), speaker: 'You', state: 'sent' });
    expect(you.find('.md-stub').exists()).toBe(false);
    expect(you.text()).toContain('plain');
  });

  it('shows the selection quote only when present', () => {
    expect(mountTurn({ message: msg(), speaker: 'You', state: 'sent' }).find('[data-test="turn-quote"]').exists()).toBe(false);
    const q = mountTurn({ message: msg({ selection_text: 'FTS5 snippet()' }), speaker: 'You', state: 'sent' });
    expect(q.find('[data-test="turn-quote"]').text()).toContain('FTS5 snippet()');
  });

  it('renders each delivery state', () => {
    expect(mountTurn({ message: msg(), speaker: 'You', state: 'sent' }).find('[data-test="turn-state"]').text()).toMatch(/sent/i);
    expect(mountTurn({ message: msg(), speaker: 'You', state: 'delivered' }).find('[data-test="turn-state"]').text()).toMatch(/Claude Code has it/i);
    expect(mountTurn({ message: msg(), speaker: 'You', state: 'undelivered' }).find('[data-test="turn-state"]').text()).toMatch(/not delivered/i);
  });

  it('turns the ribbon destructive when undelivered', () => {
    const w = mountTurn({ message: msg(), speaker: 'You', state: 'undelivered' });
    expect(w.find('[data-test="turn-ribbon"]').classes().join(' ')).toContain('bg-destructive-fg');
  });

  it('emits resend from the undelivered action and copy from any turn', async () => {
    const w = mountTurn({ message: msg(), speaker: 'You', state: 'undelivered' });
    await w.find('[data-test="turn-resend"]').trigger('click');
    expect(w.emitted('resend')).toBeTruthy();
    await w.find('[data-test="turn-copy"]').trigger('click');
    expect(w.emitted('copy')).toBeTruthy();
  });
});
