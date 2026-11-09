# Style guide

The look is called **Lanyard**. Source of truth: `apps/web/src/styles.css` (tokens) and
`apps/web/src/components/ui/*`. If this file and the code disagree, fix whichever is wrong in
the same PR.

## 1. Brand and feel

Friendly and welcoming, like the hallway track. Light, green-tinted surfaces, forest green for
actions, marigold for anything the app wants you to notice (a corrected word, a search match),
generous rounding and pill-shaped buttons. The content (live captions, search results) is the
interface; chrome stays quiet so the captions can be large and easy to read.

Accessibility is part of the look, not a layer on top. Every colour pair below was chosen to
pass WCAG AA, and captions and body text pass AAA.

| Do                                                       | Don't                                    |
| -------------------------------------------------------- | ---------------------------------------- |
| Whitespace and alignment for hierarchy                   | Boxes inside boxes, heavy dividers       |
| Semantic tokens (`bg-primary`, `text-muted-foreground`)  | Tailwind palette colours, hex literals   |
| One primary (green) action per view                      | Several solid buttons competing          |
| `bg-highlight` only for what the app changed or matched  | Marigold as decoration or for headings   |
| `border` hairlines, `shadow-xs` / `shadow-sm`            | Large drop shadows, gradients, glows     |
| Short, plain copy in sentence case                       | Title Case Everywhere, exclamation marks |
| `tabular-nums` and `font-mono` for timestamps and counts | Proportional digits in data columns      |

## 2. Colour tokens

Defined as OKLCH in `:root` and `.dark`, mapped to Tailwind utilities in `@theme inline`. Every
neutral carries a slight green tint (hue 128–140) so greys never look unconsidered. Dark mode is
the `.dark` class on `<html>`. It follows `prefers-color-scheme` until someone picks Light or
Dark in the header's `ThemeToggle`, which stores the choice in `localStorage` (`theme`). An
inline script in `index.html` applies it before first paint so there is no flash; keep it in
step with `src/lib/theme.ts`.

| Token                                | Light                 | Dark                  | Use                                          |
| ------------------------------------ | --------------------- | --------------------- | -------------------------------------------- |
| `background` / `foreground`          | `#f4f7f1` / `#1a2318` | `#10150e` / `#eef2ea` | Page and body text                           |
| `card` / `card-foreground`           | white / `#1a2318`     | `#192016` / `#eef2ea` | Card surfaces, the caption pane              |
| `popover` / `popover-foreground`     | white / `#1a2318`     | `#192016` / `#eef2ea` | Menus, popovers, dialogs                     |
| `primary` / `primary-foreground`     | `#2d683c` / white     | `#8fcb8f` / `#10200f` | Main buttons, links, active states           |
| `secondary` / `secondary-foreground` | `#e5ecdf` / `#1a2318` | `#263022` / `#eef2ea` | Secondary buttons, chips                     |
| `muted` / `muted-foreground`         | `#e9eee4` / `#556050` | `#222b1f` / `#a2ad9b` | Subtle fills, meta text                      |
| `accent` / `accent-foreground`       | `#e5ecdf` / `#1a2318` | `#263022` / `#eef2ea` | Hover and selected rows                      |
| `highlight` / `highlight-foreground` | `#ffe08a` / `#1a2318` | `#4a3b0e` / `#ffe08a` | Corrected caption terms, search term matches |
| `destructive`                        | `#b42318`             | `#f2685a`             | Errors and destructive actions               |
| `live` / `live-foreground`           | `#c92035` / white     | `#f97676` / `#250e0e` | The "Live" badge on talks being captioned    |
| `border`                             | `#d7ded0`             | `#2b3527`             | Hairlines, card and outline-button borders   |
| `input`                              | `#7f8a79`             | `#66725f`             | Form field borders                           |
| `ring`                               | `#2d683c`             | `#8fcb8f`             | Focus rings                                  |

The hex values are what the OKLCH values were derived from; change the OKLCH in `styles.css`
and update this table together.

### Contrast

Measured with the WCAG 2 formula. Keep these at or above the target when changing a colour.

| Pair                                   | Light  | Dark   | Target             |
| -------------------------------------- | ------ | ------ | ------------------ |
| `foreground` on `background`           | 15.0:1 | 16.3:1 | 7:1 (AAA)          |
| `muted-foreground` on `background`     | 6.1:1  | 7.9:1  | 4.5:1 (AA)         |
| `muted-foreground` on `muted`          | 5.6:1  | 6.3:1  | 4.5:1              |
| `primary-foreground` on `primary`      | 6.7:1  | 9.0:1  | 4.5:1              |
| `primary` on `background` (links)      | 6.2:1  | 9.8:1  | 4.5:1              |
| `highlight-foreground` on `highlight`  | 12.6:1 | 8.5:1  | 7:1 (caption text) |
| `destructive` on `background` / `card` | 6.1:1  | 5.5:1  | 4.5:1              |
| `live-foreground` on `live`            | 5.6:1  | 6.9:1  | 4.5:1              |
| `input` on `card` and `background`     | 3.3:1+ | 3.3:1+ | 3:1 (non-text)     |
| `ring` on `background`                 | 6.2:1  | 9.8:1  | 3:1 (non-text)     |

Missing for now: status colours (success, warning, info), and the `chart-*` and `sidebar-*`
tokens. Add them to `styles.css`, this table and the
contrast table when first needed.

## 3. Shape and depth

| Token                     | Value            | Use                                                |
| ------------------------- | ---------------- | -------------------------------------------------- |
| `--radius` (`rounded-lg`) | `0.75rem` (12px) | `rounded-sm` / `-md` / `-xl` are 8px / 10px / 16px |
| `rounded-xl`              | 16px             | Cards, dialogs, the caption pane                   |
| `rounded-md`              | 10px             | Inputs, menu items, search result rows             |
| `rounded-full`            |                  | Every button size and variant, badges, avatars     |
| `shadow-xs`               |                  | Outline buttons, inputs                            |
| `shadow-sm`               |                  | Cards                                              |
| `shadow-lg`               |                  | Dialogs, popovers                                  |

## 4. Typography

Two self-hosted variable fonts from Fontsource (`@fontsource-variable/lexend` and
`@fontsource-variable/red-hat-mono`), imported in `styles.css`. Self-hosting keeps the app
working offline and sends no visitor data to a font CDN.

- **Lexend** is `font-sans`, used for everything. It was designed to reduce visual stress and
  improve reading speed, which matters for people following captions for an hour.
- **Red Hat Mono** is `font-mono`, for timestamps, IDs, counts and service values.

| Element               | Spec                                                       |
| --------------------- | ---------------------------------------------------------- |
| Page title            | `text-2xl font-semibold tracking-tight`, one `h1` per page |
| Section title         | `text-lg font-semibold`                                    |
| Card title            | `leading-none font-semibold`                               |
| Body                  | `text-sm`, `text-base` for long reading                    |
| Meta and descriptions | `text-sm text-muted-foreground`                            |
| Labels                | `text-sm font-medium`                                      |
| Timestamps            | `font-mono tabular-nums`                                   |
| Code, IDs, values     | `font-mono text-sm`                                        |

### Live captions

Captions are the exception to "small and quiet".

| Property         | Spec                                                                                                                                                               |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Size and leading | `text-caption` (1.25rem / 20px, line height 1.6), larger on wide screens is fine                                                                                   |
| Weight           | `font-normal`; never `font-light`                                                                                                                                  |
| Colour           | `text-card-foreground` on `bg-card`; never `text-muted-foreground`                                                                                                 |
| Older lines      | May drop to `text-muted-foreground` once a newer line is complete (still AA)                                                                                       |
| Corrected terms  | `bg-highlight text-highlight-foreground rounded-sm px-0.5`, with the heard word as accessible text (for example a `title` plus visually hidden "corrected from …") |
| Line length      | `max-w-[65ch]` or narrower                                                                                                                                         |
| Updates          | The caption region is `aria-live="polite"`, announcing finished lines only, not every partial word                                                                 |

## 5. Layout

| Item          | Value                                                                                |
| ------------- | ------------------------------------------------------------------------------------ |
| Content width | `max-w-xl` for single-column pages today; `max-w-5xl` or `max-w-6xl` for wider views |
| Page padding  | `p-8` desktop, `p-4` phone                                                           |
| Gaps          | `gap-6` between cards, `gap-4` inside them, `gap-2` between a label and its control  |
| Card padding  | `py-6` with `px-6` on each part (stock `Card`)                                       |

## 6. Components

All UI is built from shadcn/ui components in `apps/web/src/components/ui/`. See the
`ui-components` skill for how to add one. Where this repo differs from stock shadcn
(`new-york`), the difference is listed here.

| Component         | File                   | Primitive                             | Variants / notes                                                                                                                                                                                                                                                        |
| ----------------- | ---------------------- | ------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Toggle`          | `ui/toggle.tsx`        | Radix `Toggle`                        | `default`, `outline`. Sizes `default`, `sm`, `lg`. Differs from stock: `rounded-full`, the solid offset focus ring, and the on state is `bg-primary text-primary-foreground` (stock `bg-accent` is only 1.2:1 against white, too faint to show which item is on)        |
| `ToggleGroup`     | `ui/toggle-group.tsx`  | Radix `ToggleGroup`                   | `ToggleGroup`, `ToggleGroupItem`; takes the `Toggle` variants and a `spacing`. With `spacing={0}` it is a pill-shaped segmented control. Use `type="single"` for a choice of one (it renders as a radio group)                                                          |
| `ThemeToggle`     | `theme-toggle.tsx`     | `ToggleGroup`                         | Light, Dark, Match system. Icon-only, each item labelled with `aria-label` and `title`. Uses `useTheme` (`hooks/use-theme.ts`)                                                                                                                                          |
| `Button`          | `ui/button.tsx`        | native `button`, `Slot` for `asChild` | `default`, `destructive`, `outline`, `secondary`, `ghost`, `link`. Sizes `default`, `xs`, `sm`, `lg`, `icon`, `icon-xs`, `icon-sm`, `icon-lg`. Use `asChild` to render a router `Link`. Differs from stock: `rounded-full` at every size, and a solid offset focus ring |
| `Card`            | `ui/card.tsx`          | native `div`                          | `Card`, `CardHeader`, `CardTitle`, `CardDescription`, `CardAction`, `CardContent`, `CardFooter`. Stock; `rounded-xl` is 16px here                                                                                                                                       |
| `Badge`           | `ui/badge.tsx`         | native `span`, `Slot` for `asChild`   | `default`, `secondary`, `destructive`, `outline`, `ghost`, `link`, and `live` (added here, on the `live` token). Differs from stock: the solid offset focus ring                                                                                                        |
| `Input`           | `ui/input.tsx`         | native `input`                        | Differs from stock: the solid offset focus ring                                                                                                                                                                                                                         |
| `LiveBadge`       | `live-badge.tsx`       | `Badge`                               | `live` badge with a dot that pulses only under `motion-safe`. The word "Live" carries the meaning, not the colour                                                                                                                                                       |
| `TalkCard`        | `talk-card.tsx`        | `Card`                                | A talk in a list. The title link stretches over the whole card (`after:absolute after:inset-0`), so give anything else clickable inside it `relative`. `children` render below the meta line (search highlights)                                                        |
| `HighlightedText` | `highlighted-text.tsx` | native `mark`                         | Renders an Elasticsearch highlight fragment, turning `<mark>` into `bg-highlight` without parsing any other markup                                                                                                                                                      |

## 7. Motion and accessibility

- Motion comes from `tw-animate-css` (fade and zoom on open/close) and stays short and subtle.
  Respect `prefers-reduced-motion` with `motion-safe:` / `motion-reduce:`.
- Focus is always visible and meets 3:1: `focus-visible:ring-[3px] focus-visible:ring-ring
  focus-visible:ring-offset-2 focus-visible:ring-offset-background`. Unlike stock shadcn, don't
  fade the ring to `ring-ring/50`; at half strength it drops to 2.2:1 on the light background.
  The offset keeps the ring visible around a green button. The global outline is `outline-ring`.
- Text meets WCAG AA contrast against its surface, and captions meet AAA. `muted-foreground`
  is for secondary text only.
- Never use colour alone to carry meaning: a highlighted term also has text that says why, and
  errors say what went wrong.
- Every interaction works by keyboard; Radix primitives provide this, so don't replace them
  with hand-rolled `div`s.
- Icons that carry meaning get an accessible name (`aria-label` or visually hidden text).
