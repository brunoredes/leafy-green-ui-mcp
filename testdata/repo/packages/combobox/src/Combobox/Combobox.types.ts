/**
 * Props for Combobox. Generic over whether multiple values can be selected.
 */
export interface ComboboxProps<M extends boolean> extends DarkModeProps {
  /** Allows selecting multiple options, rendered as chips. */
  multiselect?: M;

  /** Visible label for the input. Required unless `aria-label` is provided. */
  label?: string;

  /** Accessible name used when no visible label is rendered. */
  'aria-label'?: string;

  /** Callback fired when the selection changes. */
  onChange?: (value: M extends true ? Array<string> : string | null) => void;

  overflow?: { kind: 'expand-y' | 'scroll-x' };
}

export interface ComboboxOptionProps {
  /** Option value. */
  value: string;
  /** Text shown instead of the value. */
  displayName?: string;
}
