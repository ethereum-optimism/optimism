import Vue from "vue";
import AutoIcon from "@theme/icons/AutoIcon.vue";
import DarkIcon from "@theme/icons/DarkIcon.vue";
import LightIcon from "@theme/icons/LightIcon.vue";
import { changeClass } from "@theme/utils/dom";
export default Vue.extend({
    name: "DarkmodeSwitch",
    components: { AutoIcon, DarkIcon, LightIcon },
    data: () => ({
        darkmode: "auto",
    }),
    computed: {
        darkmodeConfig() {
            return this.$themeConfig.darkmode || "auto-switch";
        },
    },
    mounted() {
        this.darkmode =
            localStorage.getItem("darkmode") ||
                "auto";
        if (this.darkmodeConfig === "auto-switch")
            if (this.darkmode === "auto")
                this.setDarkmode("auto");
            else
                this.setDarkmode(this.darkmode);
        else if (this.darkmodeConfig === "auto")
            this.setDarkmode("auto");
        else if (this.darkmodeConfig === "switch")
            this.setDarkmode(this.darkmode);
        // disabled
        else
            this.setDarkmode("off");
    },
    methods: {
        setDarkmode(status) {
            if (status === "on")
                this.toggleDarkmode(true);
            else if (status === "off")
                this.toggleDarkmode(false);
            else {
                const isDarkMode = window.matchMedia("(prefers-color-scheme: dark)").matches;
                const isLightMode = window.matchMedia("(prefers-color-scheme: light)").matches;
                window
                    .matchMedia("(prefers-color-scheme: dark)")
                    .addEventListener("change", (event) => {
                    if (event.matches)
                        this.toggleDarkmode(true);
                });
                window
                    .matchMedia("(prefers-color-scheme: light)")
                    .addEventListener("change", (event) => {
                    if (event.matches)
                        this.toggleDarkmode(false);
                });
                if (isDarkMode)
                    this.toggleDarkmode(true);
                else if (isLightMode)
                    this.toggleDarkmode(false);
                else {
                    const timeHour = new Date().getHours();
                    this.toggleDarkmode(timeHour < 6 || timeHour >= 18);
                }
            }
            this.darkmode = status;
            localStorage.setItem("darkmode", status);
        },
        toggleDarkmode(isDarkmode) {
            const classes = document.body.classList;
            if (isDarkmode)
                changeClass(classes, ["theme-dark"], ["theme-light"]);
            else
                changeClass(classes, ["theme-light"], ["theme-dark"]);
        },
    },
});
//# sourceMappingURL=DarkmodeSwitch.js.map