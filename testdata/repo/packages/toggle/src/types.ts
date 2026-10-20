export interface BaseToggleProps {
  /** Whether the toggle is checked. */
  checked?: boolean;

  /**
   * Callback fired when the toggle changes.
   * @deprecated Use `onCheckedChange`.
   */
  onChange?: (checked: boolean) => void;

  /** Callback fired when the checked state changes. */
  onCheckedChange?: (checked: boolean) => void;
}

export type ToggleProps = Either<BaseToggleProps & DarkModeProps, 'aria-label' | 'aria-labelledby'>;
