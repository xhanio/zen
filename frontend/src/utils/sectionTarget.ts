// The section "being read" is the last one whose top edge has crossed a line a
// third of the way down the scroll pane. A third, not the top edge: a section
// whose heading has only just appeared is not yet the one you are reading.
export const SCROLL_ANCHOR_RATIO = 0.33;

/**
 * scrollTargetIndex picks the section the reader is on, given each section's
 * viewport top (getBoundingClientRect().top) and the pane's viewport geometry.
 *
 * Pure on purpose: the geometry is the caller's problem, the rule is testable
 * without a DOM.
 */
export function scrollTargetIndex(
  sectionTops: number[],
  paneTop: number,
  paneHeight: number,
): number {
  if (sectionTops.length === 0) return -1;
  const line = paneTop + paneHeight * SCROLL_ANCHOR_RATIO;
  let current = 0;
  sectionTops.forEach((top, i) => {
    if (top <= line) current = i;
  });
  return current;
}
