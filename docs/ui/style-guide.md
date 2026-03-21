# UI Style Guide

This document describes the visual design system for the GoShip codebase.
It is intended as a reference for developers and AI agents before editing any UI file.

Last updated: 2026-03-21

---

## 1. Theme Tokens

GoShip uses two DaisyUI themes: `lightmode` (extends `cmyk`) and `darkmode` (extends `business`).
The active theme is set via `data-theme` on the `<html>` element.

| Token | Light value + description | Dark value + description |
|---|---|---|
| `primary` | `white` — navbar and drawer background surface | `#111827` — near-black navy, navbar and drawer background |
| `secondary` | `#DEFBFB` — very pale cyan, used for subtle info banners | `#222833` — dark blue-grey, used for card surfaces |
| `accent` | `#FA6A7D` — rose pink, primary CTA highlight color | `#FA6A7D` — same rose pink, unchanged across themes |
| `neutral` | `#919191` — mid-grey, muted text and borders | `#494949` — dark grey, muted text and borders |
| `base-100` | `""` (inherits cmyk default) — base page background | `#010D14` — near-black blue, page background |
| `base-200` | cmyk default — slightly darker surface than base-100 | business default — slightly lighter than base-100 |
| `base-300` | cmyk default — used for dividers and input backgrounds | business default — used for dividers and input backgrounds |
| `info` | `#623CEA` — violet-purple, informational accents | `#623CEA` — same violet-purple, unchanged |
| `success` | `#87FF65` — bright lime green, success states | `#80D569` — muted green, success states |
| `warning` | `#FFC759` — amber, warning states | `#FFC759` — same amber, unchanged |
| `error` | `#A30000` — dark red, error states | `#A30000` — same dark red, unchanged |

> `primary-content`, `secondary-content`, `accent-content`, etc. are derived automatically by DaisyUI from the base token.

Framework-owned CSS variable layer:

- `styles/styles.css` defines the canonical `--gs-*` tokens for background, surface, text, border, accent, success, danger, radius, spacing, and shadow.
- `styles/tailwind_components.css` defines the framework-owned recipe classes that are safelisted into the bundle: `gs-page`, `gs-panel`, `gs-title`, `gs-text`, `gs-button`, `gs-button-primary`, `gs-button-secondary`, `gs-field-error`, and `gs-field-success`.
- Prefer those `gs-*` recipe classes when the framework should own the default presentation across templ pages and islands; use raw utility classes only for app-specific one-offs.

---

## 2. Typography

### Font families

| Font | Declaration | Where applied |
|---|---|---|
| `Playfair Display` | Loaded via Google Fonts (`ital,wght@0,400..900;1,400..900`). Registered in Tailwind as `font-PlayfairDisplay`. | Decorative headings and brand moments where a serif is desired. Apply with `font-PlayfairDisplay`. |
| System sans-serif | Default Tailwind/DaisyUI stack (inherited from theme). | All body text, UI labels, buttons, navigation — the default everywhere no font class is set. |
| `font-mono` | Tailwind default monospace stack. | App name wordmark in navbar and drawer top bar (`font-mono` class). |

### Usage notes

- Body text, labels, and interactive elements use the default sans-serif. No class is needed.
- Use `font-PlayfairDisplay` sparingly for large display headings or hero text.
- The `font-mono` class appears on the logo/brand name text.

---

## 3. Dark Mode Mechanism

### How it works

Dark mode is controlled by two parallel mechanisms that must stay in sync:

1. **DaisyUI theme** — `data-theme="lightmode"` or `data-theme="darkmode"` on `<html>`. This drives all DaisyUI semantic tokens (`bg-primary`, `text-primary-content`, etc.).
2. **Tailwind `dark:` prefix** — The `dark` class on `<html>`. This enables `dark:` variant utility overrides (e.g., `dark:bg-gray-800`).

### Initialization (FOUC prevention)

The `darkModeSwitcher` script in `core.templ` runs inline in `<head>` before paint:

```js
if (localStorage.getItem('color-theme') === 'darkmode' || ...) {
    document.documentElement.classList.add('dark');
    document.documentElement.setAttribute('data-theme', 'darkmode');
    document.documentElement.style.setProperty('--brightness-hover', 'var(--brightness-hover-dark)');
} else {
    document.documentElement.classList.remove('dark');
    document.documentElement.setAttribute('data-theme', 'lightmode');
    document.documentElement.style.setProperty('--brightness-hover', 'var(--brightness-hover-light)');
}
```

### User toggle

The `ThemeToggle` component (`theme_toggle.templ`) renders a Svelte component that writes `color-theme` to `localStorage` and updates `data-theme` and the `dark` class in real time. There are two toggle instances: one in the navbar (`#navbar-theme-toggle`) and one in the drawer (`#drawer-theme-toggle`).

### CSS custom properties for hover effects

`styles.css` defines brightness variables used for hover darkening/lightening:

```css
:root {
  --brightness-normal: 1;
  --brightness-hover-light: 0.15;
  --brightness-hover-dark: 0.3;
}
```

The `.hover-brightness` utility class applies a `linear-gradient` overlay on hover using `--brightness-hover`, which is set dynamically to the light or dark value.

### Writing dark-mode-aware styles

- **Prefer DaisyUI semantic tokens** (`bg-primary`, `text-base-content`) — they respond to `data-theme` automatically with no extra classes.
- **Use `dark:` prefix** for overrides where semantic tokens don't apply directly (e.g., `dark:bg-gray-700`, `dark:text-white`).
- Both patterns coexist throughout the codebase. Do not rely on only one.

---

## 4. Responsive Breakpoints

GoShip uses Tailwind's default breakpoints with a **mobile-first** approach.

| Breakpoint | Min-width | Primary use |
|---|---|---|
| (default) | 0px | Mobile layout — single column, compact spacing |
| `sm:` | 640px | Minor text scaling, occasional layout adjustments |
| `md:` | 768px | Occasional layout shifts (e.g., navbar items appear, footer switches to flex row) |
| `lg:` | 1024px | **Primary desktop breakpoint.** Sidebar/drawer becomes always-visible. Major layout changes occur here. |
| `xl:` | 1280px | Rare — minor size tweaks |

### Common responsive patterns

- **Hamburger menu vs. always-on drawer:** The drawer uses Alpine.js `isOpen: window.innerWidth >= 1024` and `@resize.window`. The hamburger toggle button has `class="... lg:hidden"` — it disappears at `lg`.
- **Hidden on mobile, shown on desktop:** `hidden md:flex` (navbar desktop links), `hidden lg:block` (text labels next to icons, e.g., "Docs" label).
- **Text size scaling:** Headings scale across breakpoints, e.g., `text-xl sm:text-2xl md:text-3xl`.
- **Drawer is fixed on mobile, static on desktop:** The drawer panel uses `fixed top-0 lg:top-auto left-0 lg:left-auto`.

---

## 5. Recurring Layout Patterns

The following patterns appear across many pages. Class strings are copied verbatim — do not abstract them.

### Pattern 1: Page container (navbar, general content)

Used for horizontal centering with padding inside a full-width bar:

```html
<div class="container mx-auto px-4 py-2 flex justify-between items-center">
```

Used inside the navbar to align logo, nav links, and user controls.

### Pattern 2: Card / feed item

Used for individual content cards in the home feed:

```html
<div class="bg-slate-100 dark:bg-slate-700 rounded-xl my-4 p-5 flex justify-center items-center">
```

Light mode: light grey card. Dark mode: slate-700 card. Rounded corners, vertical margin, internal padding.

### Pattern 3: Sidebar / drawer panel

The left sidebar that holds navigation:

```html
<div class="fixed top-0 lg:top-auto left-0 lg:left-auto z-50 h-screen p-4 overflow-y-auto
    -translate-x-full lg:translate-x-0 transform transition duration-150
    bg-primary text-primary-content w-64">
```

- `bg-primary` / `text-primary-content` — picks up the correct surface color from the active theme.
- Slides in from left on mobile (`-translate-x-full` toggled by Alpine), always visible at `lg`.
- Width is fixed at `w-64`.

### Pattern 4: Form input

Standard text input used across login, register, and other forms:

```html
<input class="bg-gray-50 border border-gray-300 text-gray-900 text-sm md:text-base rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full md:ps-5 p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500">
```

Uses `dark:` prefixes explicitly — this is the standard pattern for form inputs throughout the app.

### Pattern 5: Pill / tab button group

Used in the home feed for switching between views (Feed, Conversations, Drafts, etc.):

```html
<button class="bg-gray-400 dark:bg-gray-500 hover:bg-gray-400 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-100 font-bold py-2 px-4 rounded-full inline-flex items-center m-2">
```

Active state uses `bg-gray-400 dark:bg-gray-500`. Inactive state uses `bg-gray-300 dark:bg-gray-800`. The active/inactive swap is done at runtime with Hyperscript (`_="on click ..."`).

---

## 6. HTMX Swap Patterns

The following `hx-swap` values appear in the codebase.

### `outerHTML show:window:top`

```html
hx-swap="outerHTML show:window:top"
```

The most common full-page navigation pattern. Replaces the entire `#main-content` element and scrolls the window to the top. Used on the navbar, drawer, and footer navigation links. Always paired with:

```html
hx-target="#main-content"
hx-select="#main-content"
hx-push-url="true"
```

### `outerHTML`

```html
hx-swap="outerHTML"
```

Replaces a specific component in place without a scroll side-effect. Used when refreshing a component that re-renders itself (e.g., `homeFeedButtonsWithCounts`).

### `innerHTML`

```html
hx-swap="innerHTML"
```

Updates only the inner content of a target element. Used when loading a section into a container that must persist (e.g., loading a tab's content into `#homeFeedItems`, or updating a notification count badge).

### `beforeend swap:outerHTML`

```html
hx-swap="beforeend swap:outerHTML"
```

Used for infinite scroll pagination. A sentinel `<div>` triggers on `intersect once`, appends new items (`.temporalized-home-feed`) to the list, and then the sentinel itself is swapped away via `swap:outerHTML`.

### `hx-push-url="false"`

Used whenever the swap should not update the browser URL (e.g., inline notification count updates, tab content loads within a page that has already been navigated to).

---

## 7. Component Libraries Available

The following UI libraries are available and already loaded globally.

| Library | Version | How loaded | Priority |
|---|---|---|---|
| **DaisyUI** | bundled via Tailwind plugin | `tailwind.config.js` | **First choice.** Use DaisyUI component classes (`btn`, `badge`, `alert`, `card`, `drawer`, etc.) and semantic tokens (`bg-primary`, `text-error`, etc.) before writing custom Tailwind. |
| **Flowbite** | 2.2.1 / 2.3.0 | CDN JS in `beforeBodyEnd` + Tailwind plugin | **Second choice.** Flowbite datepicker is explicitly loaded. Other Flowbite interactive components are available via `flowbite.min.js`. Use for components DaisyUI doesn't cover well (e.g., complex dropdowns, date pickers). |
| **Alpine.js** | 3.x | CDN (deferred) | Used for client-side interactivity (`x-data`, `x-show`, `@click`, `x-cloak`). Also loaded: `@alpinejs/collapse`, `@alpinejs/morph`, `@alpinejs/mask`, `alpine-timeago`, `alpine-tooltip`, `alpine-clipboard`. |
| **HTMX** | 1.9.10 | CDN | All server-driven UI updates. SSE extension (`sse.js`) is also loaded for real-time updates. |
| **Hyperscript** | 0.9.12 | CDN (deferred) | Used for simple DOM scripting inline on elements (the `_="on click ..."` attribute). Prefer Alpine for stateful logic; prefer Hyperscript for one-off DOM manipulations. |
| **Swiper** | 11.x | CDN (deferred) | Available for carousel/slider components. |
| **Tippy.js** | 6.x | CDN (deferred CSS) | Available for tooltips. The `alpine-tooltip` plugin wraps this. |
| **Driver.js** | 1.0.1 | CDN | Available for product tour / onboarding overlays. |

### Priority order for new UI work

1. Use a **DaisyUI** component class if one exists for the pattern.
2. Fall back to **Flowbite** for interactive widgets not in DaisyUI.
3. Write custom **Tailwind utility classes** last, only when neither library covers the need.
4. For interactivity: **Alpine.js** for stateful reactive behaviour, **Hyperscript** for simple one-shot DOM actions, **HTMX** for any server round-trip.
