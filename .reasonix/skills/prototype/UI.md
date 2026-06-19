# UI Prototype
Generate **several radically different UI variations** on a single route, switchable from a floating bottom bar. The user flips between variants in the browser, picks one (or steals bits from each), then throws the rest away.
If the question is about logic/state rather than what something looks like — wrong branch. Use [LOGIC.md](LOGIC.md).

## Two sub-shapes — strongly prefer sub-shape A
A UI prototype is much easier to judge when it's **butting up against the rest of the app** — real header, real sidebar, real data, real density. Default to sub-shape A whenever there's a plausible existing page to host the variants.

### Sub-shape A — adjustment to an existing page (preferred)
The route already exists. Variants are rendered **on the same route**, gated by a `?variant=` URL search param. The existing data fetching, params, and auth all stay — only the rendering swaps.

### Sub-shape B — a new page (last resort)
Only use when the thing being prototyped genuinely has no existing page to live inside. Create a **throwaway route** following whatever routing convention the project already uses. Same `?variant=` pattern.

## Process
### 1. State the question and pick N
Default to **3 variants**. More than 5 stops being radically different and starts being noise — cap there.

### 2. Generate radically different variants
Variants must be **structurally different** — different layout, different information hierarchy, different primary affordance, not just different colours. Three slightly-tweaked card grids isn't a UI prototype, it's wallpaper.

### 3. Wire them together
Create a single switcher component on the route that renders the active variant based on `?variant=` URL param.

### 4. Build the floating switcher
A small fixed-position bar at the bottom-centre of the screen with:
- **Left arrow** — cycles to the previous variant (wraps around).
- **Variant label** — shows the current variant key and name.
- **Right arrow** — cycles forward (wraps around).
Behaviour:
- Clicking an arrow updates the URL search param so the variant is shareable and reload-stable.
- Keyboard: `←` and `→` arrow keys also cycle (don't intercept when inputs are focused).
- Hidden in production builds — gate on dev mode check.

### 5. Hand it over
Surface the URL (and the `?variant=` keys). The user will flip through and the interesting feedback is usually **"I want the header from B with the sidebar from C"** — that's the actual design they want.

### 6. Capture the answer and clean up
Once a variant has won, write down which one and why. Then:
- **Sub-shape A** — delete the losing variants and the switcher; fold the winner into the existing page.
- **Sub-shape B** — promote the winning variant to a real route, delete the throwaway route and the switcher.
Don't leave variant components or the switcher lying around. They rot fast and confuse the next reader.

## Anti-patterns
- **Variants that differ only in colour or copy.** That's a tweak, not a prototype. Real variants disagree about structure.
- **Sharing too much code between variants.** A shared `<Header>` is fine; a shared `<Layout>` defeats the point.
- **Wiring variants to real mutations.** Read-only prototypes are fine. If a variant needs to mutate, point it at a stub.
- **Promoting the prototype directly to production.** Rewrite properly when you fold it in.