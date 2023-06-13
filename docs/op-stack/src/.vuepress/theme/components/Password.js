import { getDefaultLocale } from "@mr-hope/vuepress-shared";
import Vue from "vue";
export default Vue.extend({
    name: "Password",
    props: {
        page: { type: Boolean, default: false },
    },
    data: () => ({
        password: "",
        hasTried: false,
    }),
    computed: {
        isMainPage() {
            return this.$frontmatter.home === true;
        },
        encrypt() {
            return this.$themeLocaleConfig.encrypt || getDefaultLocale().encrypt;
        }
    },
    methods: {
        verify() {
            this.hasTried = false;
            // eslint-disable-next-line vue/require-explicit-emits
            this.$emit("password-verify", this.password);
            void Vue.nextTick().then(() => {
                this.hasTried = true;
            });
        },
    },
});
//# sourceMappingURL=Password.js.map