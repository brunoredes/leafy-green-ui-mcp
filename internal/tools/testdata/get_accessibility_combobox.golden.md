# Accessibility: @leafygreen-ui/combobox 12.1.0

## Design guidelines

### Accessibility

Always provide a visible label. Selected chips must be removable with the keyboard.

## README

### Keyboard interaction

Arrow keys move the highlight through the options. Enter selects the highlighted option, and Backspace removes the last selected chip when multiselect is enabled.

## Accessibility-related props

- `label` (ComboboxProps): Visible label for the input. Required unless `aria-label` is provided. — `src/Combobox/Combobox.types.ts`
- `aria-label` (ComboboxProps): Accessible name used when no visible label is rendered. — `src/Combobox/Combobox.types.ts`

## Observed in source

- Requires `aria-label` or `aria-labelledby`; validated at runtime by `validateAriaLabelProps`. — `src/Combobox/Combobox.tsx`

## General resources

- Accessibility foundation (Color contrast, Keyboard) → get_design_guidelines name="accessibility"
- @leafygreen-ui/a11y documents `VisuallyHidden`, `validateAriaLabelProps` → get_component name="a11y"
