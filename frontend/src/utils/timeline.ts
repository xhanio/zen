import type { CardSnapshot, Message } from '../types/entity';

// A run longer than this folds. Two or three records read fine inline; a bulk
// pass across a document would otherwise flush the conversation off screen.
export const FOLD_THRESHOLD = 3;

export type TimelineItem =
  | { kind: 'message'; at: string; id: string; message: Message }
  | { kind: 'snapshot'; at: string; id: string; snapshot: CardSnapshot }
  | {
      kind: 'snapshot-group';
      at: string;
      id: string;
      snapshots: CardSnapshot[];
      linesAdded: number;
      linesRemoved: number;
    };

// Messages and snapshots are peers: neither owns the other, and position is
// decided entirely by time. Ties break by id — both are ULIDs, so that is
// creation order.
export function buildTimeline(messages: Message[], snapshots: CardSnapshot[]): TimelineItem[] {
  const flat: TimelineItem[] = [
    ...messages.map((m): TimelineItem => ({ kind: 'message', at: m.created_at, id: m.id, message: m })),
    ...snapshots.map((s): TimelineItem => ({ kind: 'snapshot', at: s.created_at, id: s.id, snapshot: s })),
  ];

  flat.sort((a, b) => {
    if (a.at !== b.at) return a.at < b.at ? -1 : 1;
    return a.id < b.id ? -1 : 1;
  });

  const out: TimelineItem[] = [];
  let run: Extract<TimelineItem, { kind: 'snapshot' }>[] = [];

  const flush = () => {
    if (run.length === 0) return;
    if (run.length <= FOLD_THRESHOLD) {
      out.push(...run);
    } else {
      out.push({
        kind: 'snapshot-group',
        at: run[0].at,
        id: run[0].id,
        snapshots: run.map((r) => r.snapshot),
        linesAdded: run.reduce((n, r) => n + r.snapshot.lines_added, 0),
        linesRemoved: run.reduce((n, r) => n + r.snapshot.lines_removed, 0),
      });
    }
    run = [];
  };

  for (const item of flat) {
    if (item.kind === 'snapshot') {
      run.push(item);
      continue;
    }
    flush();
    out.push(item);
  }
  flush();
  return out;
}
