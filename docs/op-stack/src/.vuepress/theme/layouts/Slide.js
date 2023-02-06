import Vue from "vue";
import ClickOutside from "@theme/utils/click-outside";
import ThemeColor from "@theme/components/Theme/ThemeColor.vue";
export default Vue.extend({
    name: "Slide",
    components: { ThemeColor },
    directives: { "click-outside": ClickOutside },
    data: () => ({
        showMenu: false,
    }),
    // eslint-disable-next-line vue/no-deprecated-destroyed-lifecycle
    destroyed() {
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        document.querySelector("html").classList.remove("reveal-full-page");
        document.body.classList.remove("reveal-viewport");
        document.body.style.removeProperty("--slide-width");
        document.body.style.removeProperty("--slide-height");
    },
    methods: {
        toggle() {
            this.showMenu = !this.showMenu;
        },
        back() {
            window.history.go(-1);
            this.showMenu = false;
        },
        home() {
            void this.$router.push("/");
            this.showMenu = false;
        },
        clickOutside() {
            this.showMenu = false;
        },
    },
});
//# sourceMappingURL=Slide.js.map