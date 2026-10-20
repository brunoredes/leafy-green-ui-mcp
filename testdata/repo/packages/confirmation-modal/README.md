# Confirmation Modal

## Example

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
