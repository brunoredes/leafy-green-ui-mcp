# LeafyGreen consult: "toggle a setting on or off"

Source: leafygreen-ui repository clone (reflects its checked-out branch, which may be ahead of installed versions).
Design snapshot: mongodb.design, fetched 2026-10-01.

Candidates, best first: @leafygreen-ui/toggle

## 1. @leafygreen-ui/toggle 11.0.0

LeafyGreen UI Kit Toggle

Why it matched: `toggle` (name, description, props)
Import: no import statement found in the README examples

### Design guidance

_No design guidelines page in the snapshot._

### Key props

`export type ToggleProps = Either<BaseToggleProps & DarkModeProps, 'aria-label' | 'aria-labelledby'>;`

- `checked?: boolean` — Whether the toggle is checked.
- `onCheckedChange?: (checked: boolean) => void` — Callback fired when the checked state changes.
- `onChange?: (checked: boolean) => void` **(deprecated)** — Callback fired when the toggle changes. @deprecated Use `onCheckedChange`.

### Example

_No code example found._

### Accessibility

- [prop] `ToggleProps` requires one of `aria-label`, `aria-labelledby` (enforced by the type via `Either`).

## Gaps (do not fill these from memory)

- No mongodb.design page for @leafygreen-ui/toggle in the snapshot.
- No code example found for @leafygreen-ui/toggle.

## Next steps

- get_component_props name="toggle" for every prop before writing JSX
- get_accessibility name="toggle" for the full accessibility notes
- get_component_examples name="toggle" for more examples
