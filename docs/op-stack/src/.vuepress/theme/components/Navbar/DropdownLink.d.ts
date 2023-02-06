import Vue from "vue";
import type { NavBarConfigItem } from "@theme/utils/navbar";
declare const _default: import("vue/types/vue").ExtendedVue<Vue, {
    open: boolean;
}, {
    setOpen(value: boolean): void;
    handleDropdown(event: MouseEvent): void;
    isLastItemOfArray(item: NavBarConfigItem, array: NavBarConfigItem[]): boolean;
}, {
    dropdownAriaLabel: string;
    iconPrefix: string;
}, {
    item: NavBarConfigItem;
}>;
export default _default;
