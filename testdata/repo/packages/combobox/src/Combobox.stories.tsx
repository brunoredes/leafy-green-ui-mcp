import React from 'react';
import { Combobox, ComboboxOption } from '.';

export const Multiselect = () => (
  <Combobox label="Tags" multiselect>
    <ComboboxOption value="alpha" />
  </Combobox>
);
