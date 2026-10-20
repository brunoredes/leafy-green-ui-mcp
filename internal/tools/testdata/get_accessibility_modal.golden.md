# Accessibility: @leafygreen-ui/modal 20.0.0

## README

### Additional notes > Initial focus behavior

When a Modal opens, it automatically manages focus to ensure proper accessibility. If any child element has the `autoFocus` attribute, that element receives focus.

## Accessibility-related props

- `initialFocus` (ModalProps): Element that receives focus when the modal opens. `null` disables automatic focus. — `src/Modal/Modal.types.ts`

## General resources

- Accessibility foundation (Color contrast, Keyboard) → get_design_guidelines name="accessibility"
- @leafygreen-ui/a11y documents `VisuallyHidden`, `validateAriaLabelProps` → get_component name="a11y"
