import { DarkModeProps } from '@leafygreen-ui/lib';

export const Variant = {
  Default: 'default',
  Primary: 'primary',
  Danger: 'danger',
} as const;

export type Variant = (typeof Variant)[keyof typeof Variant];

/**
 * Props for the Button component.
 */
export interface ButtonProps extends DarkModeProps, HTMLElementProps<'button'> {
  /**
   * Sets the variant for the Button.
   * @default 'default'
   */
  variant?: Variant;

  /** Disables the Button. Uses `aria-disabled` so the button stays focusable. */
  disabled?: boolean;
}
