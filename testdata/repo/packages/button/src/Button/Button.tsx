export const Button = ({ variant, disabled, ...rest }: ButtonProps) => (
  <button aria-disabled={disabled} {...rest} />
);
