---

name: ui-components
description: Use whenever building or changing UI in apps/web (routes, pages, forms, dialogs, tables, buttons, any JSX markup). Decides whether a piece of UI should be a reusable component, copies it from the shadcn/ui registry when one exists and restyles it to this repo, and otherwise builds one the shadcn way on a Radix primitive.
---

# UI components

Stack: React 19, TanStack Router and Query, Tailwind CSS v4, shadcn/ui (`new-york` style), the
unified `radix-ui` package, `lucide-react` icons, Biome. All paths below are relative to
`apps/web`. `src/components/ui/button.tsx` and `card.tsx` are finished examples of copied
components; compare against them.

## 1. Component or inline?

Make a component for interactive primitives (dialog, menu, select, tabs, tooltip, toast…) and
for visual patterns with variants or more than one use (button, badge, card, input). Keep
one-off page layout inline in the route.

- Generic primitives go in `src/components/ui/<kebab-name>.tsx`. Domain components that compose
  them go in `src/components/<kebab-name>.tsx` and export a PascalCase component.
- Extend an existing `ui/` component with a variant before adding a new one. Don't add a second
  component for the same job (no `sonner` next to `toast`, no `native-select` next to `select`).

## 2. Look in the shadcn registry first

```sh
curl -s https://ui.shadcn.com/r/index.json | jq -r '.[].name'        # every component name
curl -s https://ui.shadcn.com/r/styles/new-york-v4/<name>.json       # source, dependencies, registryDependencies, cssVars
```

Match by concept, not just name (autocomplete → `combobox`, modal → `dialog`, side panel →
`sheet`, loading placeholder → `skeleton`). A missing entry returns HTTP 404. The registry is
already Tailwind v4, so no class conversion is needed.

## 3. Copy it

The shadcn CLI resolves this repo's `cn` import wrongly: it writes `from "cn"` and tries to
install an unrelated npm package called `cn`. Preview with
`pnpm dlx shadcn@latest add <name> --dry-run --view` if useful, but copy by hand:

1. Write `files[].content` into `src/components/ui/` (`registry:hook` items into `src/hooks/`).
   Repeat for each `registryDependencies` entry not already there. `pnpm add` any missing
   `dependencies`, except `cn` (that's `~/lib/utils`) and anything already in `package.json`.
2. Rewrite imports: `"cn"` → `"~/lib/utils"`, `@/registry/new-york-v4/ui/<x>` →
   `~/components/ui/<x>`, `@/registry/new-york-v4/hooks/<x>` → `~/hooks/<x>`.
3. If the item has `cssVars` (chart, sidebar), add those tokens to `src/styles.css` in both
   `:root` and `.dark`, map them in `@theme inline`, and list them in the style guide.
4. Restyle per step 5, then `pnpm check:fix`.

Say which dependency components you pulled in.

## 4. No match: build it the shadcn way

Before writing any behavior (open state, focus management, keyboard navigation, ARIA,
positioning, portals), find the Radix primitive for it in the
[Radix Primitives docs](https://www.radix-ui.com/primitives/docs/overview/introduction) and import it
from `radix-ui` (`import { Popover as PopoverPrimitive } from "radix-ui"`). Hand-write behavior
only when Radix has nothing, and say so.

Mirror `button.tsx` and the closest shadcn source:

- Plain function components typed with `React.ComponentProps<"div">` or
  `React.ComponentProps<typeof XPrimitive.Root>`. No `forwardRef`; React 19 passes `ref` as a prop.
- Destructure `className` and spread `...props` onto the root. Set `data-slot="<part-name>"`
  on every part.
- Variants in an exported `cva` `<name>Variants` const, merged with
  `cn(nameVariants({ variant, size, className }))`.
- `asChild` via `Slot.Root` from `radix-ui` when the part should be able to render as a link
  (`<Button asChild><Link to="/">…</Link></Button>`).
- Compound parts as named exports from one file (`Card`, `CardHeader`, `CardContent`).
- Style state off data and ARIA attributes (`data-[state=open]:`, `aria-invalid:`), not
  computed class strings.

## 5. Style it

Follow `docs/style-guide.md`, using the tokens in `src/styles.css`: semantic utilities
(`bg-background`, `text-muted-foreground`, `border-input`, `ring-ring`…) only. No Tailwind
default palette colours (`zinc-*`, `blue-*`, `white`, `black`), no colour literals, and no
new arbitrary values in app code (`rounded-[12px]` → `rounded-lg`). If a token is missing, add it
to both files. Semantic tokens switch with `.dark`, so app code shouldn't need `dark:`. The
`dark:` and arbitrary classes that shadcn components ship with stay as they are.

Restyle a copied component's classes only where the style guide says this repo differs from
stock shadcn; keep its structure, props, `data-slot`s and variant names.

## 6. Use and verify

Replace the ad-hoc markup at the call site, and mention other places that could use the
component rather than migrating them. Add it to the style guide's components table. Then
`pnpm check`, `pnpm typecheck` and `pnpm test` must pass. Look at it rendered (`pnpm dev`, or
`pnpm dev:web` for the frontend alone, at http://localhost:4000) in every variant and state, at
desktop and phone widths, and with the keyboard alone. Report anything you couldn't check.
