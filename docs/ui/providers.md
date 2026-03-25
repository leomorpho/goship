# UI Providers

## What a UI Provider Is
A UI provider is a scaffold-time config value that tells GoShip which CSS/JS assets the generated layout shell should load.

The provider is selected when you create a project and is written into scaffold output.

## Available Providers
- `franken`: FrankenUI + UIkit assets loaded by the base layout.
- `daisy`: DaisyUI-oriented scaffold with Flowbite asset loading.
- `bare`: plain Tailwind-oriented scaffold with no external UI library assets.

## How To Choose
Use `ship new --ui <provider>` when creating a project.

If you do not pass `--ui`, the default is `franken`.

## Generated Output Shape
Generated view code is intentionally structural:
- bare HTML structure
- HTMX wiring (`hx-*`)
- `data-*` hooks for component/slot/action discovery

GoShip does not generate UI-library-specific classes in scaffolded page templates.

## Contract Boundary
The provider contract is generation-time only.

GoShip guarantees provider-correct output for freshly generated files.
After you edit generated files, those edits are application-owned. Automatic provider conversion for hand-edited code is explicitly not supported.

## Admin Module Exception
The admin module is self-contained and not part of the UI provider system contract.
