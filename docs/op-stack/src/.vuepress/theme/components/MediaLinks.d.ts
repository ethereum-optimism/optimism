import Vue from "vue";
import type { BlogMedia } from "@theme/types";
interface MediaLink {
    icon: string;
    url: string;
}
declare const _default: import("vue/types/vue").ExtendedVue<Vue, unknown, unknown, {
    mediaLink: false | Partial<Record<BlogMedia, string>>;
    links: MediaLink[];
}, Record<never, any>>;
export default _default;
