import Vue from "vue";
interface ThemeColor {
    /** Color list */
    list: string[];
    /** Color picker */
    picker: Record<string, string>;
}
declare const _default: import("vue/types/vue").ExtendedVue<Vue, {
    themeColor: ThemeColor;
    isDarkmode: boolean;
}, {
    setTheme(theme?: string | undefined): void;
}, {
    text: {
        themeColor: string;
        themeMode: string;
    };
    themeColorEnabled: boolean;
    switchEnabled: boolean;
}, Record<never, any>>;
export default _default;
