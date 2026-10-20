export interface ConfirmationModalProps extends Omit<ModalProps, 'setOpen'> {
  /** Text of the modal title. */
  title: string;

  /** Variant of the modal; `danger` renders a destructive confirm button. */
  variant?: 'default' | 'danger';

  /** If set, the user must type this text before confirming. */
  requiredInputText?: string;
}
