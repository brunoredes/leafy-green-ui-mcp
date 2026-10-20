# LeafyGreen consult: "let users select multiple tags from a long list"

Source: leafygreen-ui repository clone (reflects its checked-out branch, which may be ahead of installed versions).
Design snapshot: mongodb.design, fetched 2026-10-01.

Candidates, best first: @leafygreen-ui/combobox

## 1. @leafygreen-ui/combobox 12.1.0

LeafyGreen UI Kit Combobox

Why it matched: `select` (design guidance, README, design docs, props); `multiple` (design guidance, README, props); `tags` (design guidance); `long` (design guidance); `list` (design guidance); `multiselect` (README, props); `chip` (README, design docs, props)
Import: `import { Combobox, ComboboxOption } from '@leafygreen-ui/combobox';`
Peers: @leafygreen-ui/leafygreen-provider

### Design guidance

**How to use this component:** Use a combobox when the list has more than about ten options, or when users need to select multiple tags. For short lists, use a Select instead.

**Overview:** Comboboxes let users choose one or several options from a long list by typing to filter.

Source: https://www.mongodb.design/component/combobox/design-docs

### Key props (ComboboxProps<M extends boolean>, extends `DarkModeProps`)

- `multiselect?: M` — Allows selecting multiple options, rendered as chips.
- `label?: string` — Visible label for the input. Required unless `aria-label` is provided.
- `aria-label?: string` — Accessible name used when no visible label is rendered.
- `onChange?: (value: M extends true ? Array<string> : string | null) => void` — Callback fired when the selection changes.
- `overflow?: { kind: 'expand-y' | 'scroll-x' }`

### Example (README › Example)

```tsx
import { Combobox, ComboboxOption } from '@leafygreen-ui/combobox';

<Combobox label="Choose fruits" multiselect={true} placeholder="Select tags">
  <ComboboxOption value="apple" />
  <ComboboxOption value="banana" />
</Combobox>;
```

### Accessibility

- [Design guidelines › Accessibility] Always provide a visible label. Selected chips must be removable with the keyboard.
- [README › Keyboard interaction] Arrow keys move the highlight through the options. Enter selects the highlighted option, and Backspace removes the last selected chip when multiselect is enabled.
- [prop] `label` (ComboboxProps): Visible label for the input. Required unless `aria-label` is provided.
- [prop] `aria-label` (ComboboxProps): Accessible name used when no visible label is rendered.
- [source] Requires `aria-label` or `aria-labelledby`; validated at runtime by `validateAriaLabelProps`.

## Cross-cutting guidance

- `@leafygreen-ui/leafygreen-provider` is a peer dependency of combobox; render inside `LeafyGreenProvider`.

## Next steps

- get_component_props name="combobox" for every prop before writing JSX
- get_accessibility name="combobox" for the full accessibility notes
- get_component_examples name="combobox" for more examples
