export interface ModalProps extends HTMLElementProps<'dialog'> {
  /** Determines the open state of the modal. */
  open?: boolean;

  /** Callback to change the open state of the Modal. */
  setOpen?: (open: boolean) => void;

  /**
   * Element that receives focus when the modal opens. `null` disables
   * automatic focus.
   */
  initialFocus?: string | null;
}
