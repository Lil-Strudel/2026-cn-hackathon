# Style guide

> **Placeholder.** This describes stock shadcn/ui (`new-york` style, `neutral` base colour),
> which is what the app looks like today. Replace it once the team picks a product name and
> a look on the day. Keep the structure: brand, tokens, type, layout, surfaces, components,
> accessibility.

Source of truth: `apps/web/src/styles.css` (tokens) and `apps/web/src/components/ui/*`.

## 1. Brand and feel

Neutral, quiet and functional. Greyscale surfaces, near-black ink, one accent colour at most,
hairline borders, small radii and almost no shadow. The content (live captions, search
results) is the interface; chrome stays out of the way.

| Do                                                      | Don't                                    |
| ------------------------------------------------------- | ---------------------------------------- |
| Whitespace and alignment for hierarchy                  | Boxes inside boxes, heavy dividers       |
| Semantic tokens (`bg-primary`, `text-muted-foreground`) | Tailwind palette colours, hex literals   |
| One primary action per view                             | Several solid buttons competing          |
| `border` hairlines, `shadow-xs` / `shadow-sm`           | Large drop shadows, gradients, glows     |
| Short, plain copy in sentence case                      | Title Case Everywhere, exclamation marks |
| `tabular-nums` for timestamps and counts                | Proportional digits in data columns      |

## 2. Colour tokens

Defined as OKLCH in `:root` and `.dark`, mapped to Tailwind utilities in `@theme inline`. Dark
mode is the `.dark` class on an ancestor; there is no toggle yet.

| Token                                | Light                   | Dark                    | Use                               |
| ------------------------------------ | ----------------------- | ----------------------- | --------------------------------- |
| `background` / `foreground`          | white / near-black      | near-black / near-white | Page and body text                |
| `card` / `card-foreground`           | white / near-black      | dark grey / near-white  | Card surfaces                     |
| `popover` / `popover-foreground`     | white / near-black      | dark grey / near-white  | Menus, popovers, dialogs          |
| `primary` / `primary-foreground`     | near-black / near-white | light grey / dark grey  | Main buttons, active states       |
| `secondary` / `secondary-foreground` | light grey / dark grey  | dark grey / near-white  | Secondary buttons, chips          |
| `muted` / `muted-foreground`         | light grey / mid grey   | dark grey / light grey  | Subtle fills, captions, meta text |
| `accent` / `accent-foreground`       | light grey / dark grey  | dark grey / near-white  | Hover and selected rows           |
| `destructive`                        | red                     | lighter red             | Errors and destructive actions    |
| `border`, `input`                    | light grey              | white at 10–15%         | Hairlines and field borders       |
| `ring`                               | mid grey                | mid grey                | Focus rings                       |

Missing for now: status colours (success, warning, info) and the `chart-*` and `sidebar-*`
tokens. Add them to `styles.css` and this table when first needed.

## 3. Shape and depth

| Token                     | Value             | Use                                                                    |
| ------------------------- | ----------------- | ---------------------------------------------------------------------- |
| `--radius` (`rounded-lg`) | `0.625rem` (10px) | Cards and dialogs. `rounded-sm` / `-md` / `-xl` are −4px / −2px / +4px |
| `rounded-md`              | 8px               | Buttons, inputs, menu items                                            |
| `rounded-full`            |                   | Badges, avatars                                                        |
| `shadow-xs`               |                   | Outline buttons, inputs                                                |
| `shadow-sm`               |                   | Cards                                                                  |
| `shadow-lg`               |                   | Dialogs, popovers                                                      |

## 4. Typography

The system font stack (Tailwind's default `font-sans`); no web fonts.

| Element               | Spec                                    |
| --------------------- | --------------------------------------- |
| Page title            | `text-2xl font-semibold tracking-tight` |
| Section title         | `text-lg font-semibold`                 |
| Card title            | `leading-none font-semibold`            |
| Body                  | `text-sm`, `text-base` for long reading |
| Meta and descriptions | `text-sm text-muted-foreground`         |
| Labels                | `text-sm font-medium`                   |
| Code and IDs          | `font-mono text-sm`                     |

Live captions are the exception to "small and quiet": they will need a larger size, generous
line height and strong contrast. Specify them here when designed.

## 5. Layout

| Item          | Value                                                                                |
| ------------- | ------------------------------------------------------------------------------------ |
| Content width | `max-w-xl` for single-column pages today; `max-w-5xl` or `max-w-6xl` for wider views |
| Page padding  | `p-8` desktop, `p-4` phone                                                           |
| Gaps          | `gap-6` between cards, `gap-4` inside them, `gap-2` between a label and its control  |
| Card padding  | `py-6` with `px-6` on each part (stock `Card`)                                       |

## 6. Components

All UI is built from shadcn/ui components in `apps/web/src/components/ui/`. See the
`ui-components` skill for how to add one.

| Component | File            | Primitive                             | Variants / notes                                                                                                                                                                       |
| --------- | --------------- | ------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Button`  | `ui/button.tsx` | native `button`, `Slot` for `asChild` | `default`, `destructive`, `outline`, `secondary`, `ghost`, `link`. Sizes `default`, `xs`, `sm`, `lg`, `icon`, `icon-xs`, `icon-sm`, `icon-lg`. Use `asChild` to render a router `Link` |
| `Card`    | `ui/card.tsx`   | native `div`                          | `Card`, `CardHeader`, `CardTitle`, `CardDescription`, `CardAction`, `CardContent`, `CardFooter`                                                                                        |

## 7. Motion and accessibility

- Motion comes from `tw-animate-css` (fade and zoom on open/close) and stays short and subtle.
  Respect `prefers-reduced-motion` with `motion-safe:` / `motion-reduce:`.
- Focus is always visible: `focus-visible:ring-[3px] focus-visible:ring-ring/50`, as in the
  stock components.
- Text meets WCAG AA contrast against its surface. `muted-foreground` is for secondary text
  only, never for captions.
- Every interaction works by keyboard; Radix primitives provide this, so don't replace them
  with hand-rolled `div`s.
- Icons that carry meaning get an accessible name (`aria-label` or visually hidden text).
