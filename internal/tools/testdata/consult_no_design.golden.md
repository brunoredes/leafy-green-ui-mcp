# LeafyGreen consult: "open a dialog"

Source: leafygreen-ui repository clone (reflects its checked-out branch, which may be ahead of installed versions).
Design snapshot: not loaded (run `lg-mcp sync --design`).

Candidates, best first: @leafygreen-ui/modal

## 1. @leafygreen-ui/modal 20.0.0

LeafyGreen UI Kit Modal

Why it matched: `open` (README, props); `modal` (name, description, README, props)
Import: `import Modal from '@leafygreen-ui/modal';`
Peers: @leafygreen-ui/leafygreen-provider

### Design guidance

_Design snapshot not loaded._

### Key props (ModalProps, extends `HTMLElementProps<'dialog'>`)

- `open?: boolean` — Determines the open state of the modal.
- `setOpen?: (open: boolean) => void` — Callback to change the open state of the Modal.
- `initialFocus?: string | null` — Element that receives focus when the modal opens. `null` disables automatic focus.

### Example (README › Example)

```tsx
import Modal from '@leafygreen-ui/modal';

<Modal open={open} setOpen={setOpen}>
  Modal content
</Modal>;
```

### Accessibility

- [README › Additional notes > Initial focus behavior] When a Modal opens, it automatically manages focus to ensure proper accessibility. If any child element has the `autoFocus` attribute, that element receives focus.
- [prop] `initialFocus` (ModalProps): Element that receives focus when the modal opens. `null` disables automatic focus.

## Cross-cutting guidance

- `@leafygreen-ui/leafygreen-provider` is a peer dependency of modal; render inside `LeafyGreenProvider`.

## Gaps (do not fill these from memory)

- No design snapshot loaded, so design guidance is missing for every candidate. Run `lg-mcp sync --design`.

## Next steps

- get_component_props name="modal" for every prop before writing JSX
- get_accessibility name="modal" for the full accessibility notes
- get_component_examples name="modal" for more examples
