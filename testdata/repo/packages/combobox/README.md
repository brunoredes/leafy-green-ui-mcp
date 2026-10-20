# Combobox

## Installation

```shell
npm install @leafygreen-ui/combobox
```

## Example

```tsx
import { Combobox, ComboboxOption } from '@leafygreen-ui/combobox';

<Combobox label="Choose fruits" multiselect={true} placeholder="Select tags">
  <ComboboxOption value="apple" />
  <ComboboxOption value="banana" />
</Combobox>;
```

## Keyboard interaction

Arrow keys move the highlight through the options. Enter selects the highlighted option, and Backspace removes the last selected chip when multiselect is enabled.

## Properties

| Prop | Type | Description |
| --- | --- | --- |
| `multiselect` | `boolean` | Allows selecting multiple options, rendered as chips. |
| `label` | `string` | Visible label for the input. |
