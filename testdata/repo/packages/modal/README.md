# Modal

## Example

```tsx
import Modal from '@leafygreen-ui/modal';

<Modal open={open} setOpen={setOpen}>
  Modal content
</Modal>;
```

## Additional notes

### Initial focus behavior

When a Modal opens, it automatically manages focus to ensure proper accessibility. If any child element has the `autoFocus` attribute, that element receives focus.

### Backdrop styling

Use the native `::backdrop` pseudo-element.
