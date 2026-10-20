# LeafyGreen consult: "confirm a destructive delete action"

Source: leafygreen-ui repository clone (reflects its checked-out branch, which may be ahead of installed versions).
Design snapshot: mongodb.design, fetched 2026-10-01.

Candidates, best first: @leafygreen-ui/confirmation-modal, @leafygreen-ui/modal

## 1. @leafygreen-ui/confirmation-modal 8.0.0

LeafyGreen UI Kit Confirmation Modal

Why it matched: `confirm` (name, description, README, props); `destructive` (props); `danger` (props)
Import: `import ConfirmationModal from '@leafygreen-ui/confirmation-modal';`
Peers: @leafygreen-ui/leafygreen-provider

### Design guidance

_No design guidelines page in the snapshot._

### Key props (ConfirmationModalProps, extends `Omit<ModalProps, 'setOpen'>`)

- `variant?: 'default' | 'danger'` — Variant of the modal; `danger` renders a destructive confirm button.
- `requiredInputText?: string` — If set, the user must type this text before confirming.
- `title: string` — Text of the modal title.

### Example (README › Example)

```tsx
import ConfirmationModal from '@leafygreen-ui/confirmation-modal';

<ConfirmationModal
  open={open}
  title="Delete cluster"
  variant="danger"
  requiredInputText="delete"
  confirmButtonProps={{ children: 'Delete' }}
>
  This action cannot be undone.
</ConfirmationModal>;
```

### Accessibility

_No accessibility notes found._

## 2. @leafygreen-ui/modal 20.0.0

LeafyGreen UI Kit Modal

Why it matched: `confirm` (design guidance); `action` (design guidance)
Import: `import Modal from '@leafygreen-ui/modal';`
Peers: @leafygreen-ui/leafygreen-provider

### Design guidance

**Overview:** Modals can be used to display a simple task, to confirm actions, prompt users to input information, or display additional information.

**How to use this component:** Use a concise title to inform the user of what action they are about to take.

Source: https://www.mongodb.design/component/modal/design-docs

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

- `@leafygreen-ui/leafygreen-provider` is a peer dependency of confirmation-modal, modal; render inside `LeafyGreenProvider`.

## Gaps (do not fill these from memory)

- No mongodb.design page for @leafygreen-ui/confirmation-modal in the snapshot.
- No accessibility notes found for @leafygreen-ui/confirmation-modal; check the accessibility foundation guidance.

## Next steps

- get_component_props name="confirmation-modal" for every prop before writing JSX
- get_accessibility name="confirmation-modal" for the full accessibility notes
- get_component_examples name="confirmation-modal" for more examples
