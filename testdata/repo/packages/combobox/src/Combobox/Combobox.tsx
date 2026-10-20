import { validateAriaLabelProps } from '@leafygreen-ui/a11y';

export function Combobox<M extends boolean>(props: ComboboxProps<M>) {
  validateAriaLabelProps(props, 'Combobox');
  return null;
}
