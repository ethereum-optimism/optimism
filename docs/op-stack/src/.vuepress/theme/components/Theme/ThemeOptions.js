import Vue from "vue";
import { getDefaultLocale } from "@mr-hope/vuepress-shared";
import DarkmodeSwitch from "@theme/components/Theme/DarkmodeSwitch.vue";
const defaultColorPicker = {
    red: "#e74c3c",
    blue: "#3498db",
    green: "#3eaf7c",
    orange: "#f39c12",
    purple: "#8e44ad",
};
export default Vue.extend({
    name: "ThemeOptions",
    components: { DarkmodeSwitch },
    data: () => ({
        themeColor: {},
        isDarkmode: false,
    }),
    computed: {
        text() {
            return (this.$themeLocaleConfig.themeColor || getDefaultLocale().themeColor);
        },
        themeColorEnabled() {
            return this.$themeConfig.themeColor !== false;
        },
        switchEnabled() {
            return (this.$themeConfig.darkmode !== "disable" &&
                this.$themeConfig.darkmode !== "auto");
        },
    },
    mounted() {
        const theme = localStorage.getItem("theme");
        this.themeColor = {
            list: this.$themeConfig.themeColor
                ? Object.keys(this.$themeConfig.themeColor)
                : Object.keys(defaultColorPicker),
            picker: this.$themeConfig.themeColor || defaultColorPicker,
        };
        if (theme)
            this.setTheme(theme);
    },
    methods: {
        setTheme(theme) {
            const classes = document.body.classList;
            const themes = this.themeColor.list.map((colorTheme) => `theme-${colorTheme}`);
            if (!theme) {
                localStorage.removeItem("theme");
                classes.remove(...themes);
                return;
            }
            classes.remove(...themes.filter((themeclass) => themeclass !== `theme-${theme}`));
            classes.add(`theme-${theme}`);
            localStorage.setItem("theme", theme);
        },
    },
});
//# sourceMappingURL=ThemeOptions.js.map