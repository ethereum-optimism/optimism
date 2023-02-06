import Vue from "vue";
import type { NavBarConfigItem } from "@theme/utils/navbar";
declare const _default: import("vue/types/vue").ExtendedVue<Vue, unknown, {
    focusoutAction(): void;
}, {
    link: string;
    iconPrefix: string;
    active: boolean;
    isNonHttpURI: boolean;
    isBlankTarget: boolean;
    isInternal: boolean;
    target: string | null;
    rel: string | null;
}, {
    item: NavBarConfigItem;
}>;
export default _default;
