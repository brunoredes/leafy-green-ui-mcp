# Props: @leafygreen-ui/combobox 12.1.0

## ComboboxProps<M extends boolean>

interface in `src/Combobox/Combobox.types.ts` · extends `DarkModeProps`

Props for Combobox. Generic over whether multiple values can be selected.

| Prop | Type | Required | Description |
| --- | --- | --- | --- |
| `multiselect` | `M` | no | Allows selecting multiple options, rendered as chips. |
| `label` | `string` | no | Visible label for the input. Required unless `aria-label` is provided. |
| `aria-label` | `string` | no | Accessible name used when no visible label is rendered. |
| `onChange` | `(value: M extends true ? Array<string> : string \| null) => void` | no | Callback fired when the selection changes. |
| `overflow` | `{ kind: 'expand-y' \| 'scroll-x' }` | no |  |

## ComboboxOptionProps

interface in `src/Combobox/Combobox.types.ts`

| Prop | Type | Required | Description |
| --- | --- | --- | --- |
| `value` | `string` | yes | Option value. |
| `displayName` | `string` | no | Text shown instead of the value. |

Note: members inherited through `extends` are not expanded; look up those types separately if needed.
