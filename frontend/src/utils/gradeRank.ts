import type { ReviewGrade } from '../types/entity';

// Scrutiny order: '' (ungraded) < LGTM < DIGESTED < GRILLED. Higher = more
// scrutiny. Auto-transitions only ever escalate, so this ranking is the guard.
export function gradeRank(grade: ReviewGrade | ''): number {
  return ({ '': 0, LGTM: 1, DIGESTED: 2, GRILLED: 3 } as Record<string, number>)[grade] ?? 0;
}
