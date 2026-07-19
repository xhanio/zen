import { describe, it, expect } from 'vitest';
import { scrollTargetIndex, SCROLL_ANCHOR_RATIO } from './sectionTarget';

// The anchor line sits a third of the way down the pane. A section is "the one
// being read" once its top has passed that line, so the newest such section wins.
describe('scrollTargetIndex', () => {
  const paneTop = 100;
  const paneHeight = 300; // anchor line at 100 + 99 = 199

  it('is 0 when nothing has crossed the line yet', () => {
    expect(scrollTargetIndex([250, 400, 600], paneTop, paneHeight)).toBe(0);
  });

  it('picks the last section whose top crossed the line', () => {
    expect(scrollTargetIndex([-50, 120, 400], paneTop, paneHeight)).toBe(1);
  });

  it('picks the final section when all have crossed', () => {
    expect(scrollTargetIndex([-900, -600, -100], paneTop, paneHeight)).toBe(2);
  });

  it('treats a top exactly on the line as crossed', () => {
    expect(scrollTargetIndex([0, 199, 500], paneTop, paneHeight)).toBe(1);
  });

  it('returns -1 for no sections', () => {
    expect(scrollTargetIndex([], paneTop, paneHeight)).toBe(-1);
  });

  it('exposes the ratio it uses', () => {
    expect(SCROLL_ANCHOR_RATIO).toBe(0.33);
  });
});
