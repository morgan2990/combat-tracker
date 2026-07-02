## Context

A survey found 297 hex-color-literal occurrences across 24 files. A frequency count showed 10 colors accounting for ~255 of those (86%): the app's text (primary/secondary/muted), border, panel/card backgrounds, and 4 status colors (accent/orange, danger/red, info/blue, success/green). The remaining ~40 occurrences span ~30 distinct, mostly single-occurrence colors — condition-toggle active-state backgrounds, specific button hover/pressed shades, and `entityVitals.ts`'s vital-state row colors.

## Goals / Non-Goals

**Goals:**
- A single `frontend/src/theme.ts` module is the source of truth for the 10 core colors and 3 border-radius values genuinely reused across the codebase.
- Every exact-match occurrence of those 13 values is replaced with a token reference, including values embedded in composite CSS shorthand strings (e.g. `border: '1px solid #2e2e48'`).
- Zero visual change — every substituted token resolves to the exact same value it replaced.

**Non-Goals:**
- Not tokenizing every hex literal in the codebase. Colors that are single-purpose (e.g. `entityVitals.ts`'s dead/unconscious/active-turn row colors, already centralized in one file) or that coincidentally share a value with an unrelated component (e.g. `#1a1608` used both for "unconscious" and, separately, for `PlayerView.tsx`'s initiative-prompt background) are left as-is — forcing them into a shared token would create false coupling between design decisions that happen to look the same today but aren't actually the same decision.
- Not tokenizing padding/margin/gap spacing values — unlike colors and border-radius, these are highly varied and contextual (`'6px 8px'`, `'8px 16px'`, etc.), not a small fixed set worth naming.
- Not touching `entityVitals.ts`'s `vitalRowBg`/`vitalTextColor` internals beyond the two return values that already coincide with the general palette (`'#1a1a2c'` → `color.bgPanel` and `'#d4d4e8'` → `color.textPrimary`, both representing the "alive, normal" state — i.e. the app's default colors, not a distinct vital-state color).

## Decisions

**10 colors, chosen by frequency and semantic clarity, not an arbitrary round number.** Every color used 8+ times across the codebase made the cut; nothing below that threshold did. This naturally produced a coherent set: 3 text tones, 1 border, 2 backgrounds, 4 status colors — a complete enough palette to name without stretching for justification on borderline cases.

**Border-radius gets 3 tokens (`sm`/`md`/`pill`) matching the only three radii used more than twice** (4px for buttons/inputs, 6px for cards/panels, 12px for pill-shaped elements). `borderRadius: 8` (2 occurrences) and `borderRadius: 3` (1 occurrence) are left as literals — too rare to be a "shared decision" rather than a one-off.

**Composite CSS shorthand strings become template literals.** `border: '1px solid #2e2e48'` (36 occurrences) and the one `'2px solid #2e2e48'` become `` border: `1px solid ${color.border}` ``, preserving the exact rendered value while still referencing the shared token, rather than leaving these 37 occurrences as un-tokenized literals just because they're embedded in a larger string.

**Mechanical, scripted replacement rather than manual per-file edits.** Given the exact-match nature of the substitution (a given hex string always means the same color everywhere the token applies) and the volume (24 files), a script performing the same regex-based replacement across all files is lower-risk than 24 rounds of manual editing — verified afterward via `tsc -b` (would catch any malformed template literal or broken import), `oxlint`, and a visual pass confirming the rendered output is unchanged.

## Risks / Trade-offs

- [A scripted mechanical replacement misses an edge-case string format (e.g. double-quoted, different border-style keyword) and leaves a stray literal] → Verified via `grep` for each of the 10 target hex values after the script ran, confirming zero remaining occurrences outside `theme.ts` itself.
- [A composite-string conversion to a template literal introduces a typo or breaks the string] → `tsc -b` compiles clean, which would catch a malformed template literal; spot-checked several diffs by hand before running the full verification pass.

## Migration Plan

Frontend-only, no data migration. Single commit: add `theme.ts`, apply the scripted replacement across all 24 files, verify.

## Open Questions

None.
