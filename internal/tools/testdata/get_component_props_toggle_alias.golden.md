# Props: @leafygreen-ui/toggle 11.0.0

## ToggleProps

type in `src/types.ts`

```ts
export type ToggleProps = Either<BaseToggleProps & DarkModeProps, 'aria-label' | 'aria-labelledby'>;
```

## BaseToggleProps

interface in `src/types.ts`

| Prop | Type | Required | Description |
| --- | --- | --- | --- |
| `checked` | `boolean` | no | Whether the toggle is checked. |
| `onChange` | `(checked: boolean) => void` | no | Callback fired when the toggle changes. @deprecated Use `onCheckedChange`. |
| `onCheckedChange` | `(checked: boolean) => void` | no | Callback fired when the checked state changes. |

Note: members inherited through `extends` are not expanded; look up those types separately if needed.
