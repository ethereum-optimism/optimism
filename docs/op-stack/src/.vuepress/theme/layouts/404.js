import Vue from "vue";
import Common from "@theme/components/Common.vue";
import Page404Icon from "@theme/icons/Page404Icon.vue";
import { getDefaultLocale } from "@mr-hope/vuepress-shared";
export default Vue.extend({
    name: "NotFound",
    components: {
        Common,
        Page404Icon,
    },
    computed: {
        i18n() {
            return this.$themeLocaleConfig.error404 || getDefaultLocale().error404;
        },
        msg() {
            return this.i18n.hint[Math.floor(Math.random() * this.i18n.hint.length)];
        },
    },
    methods: {
        back() {
            window.history.go(-1);
        },
    },
});
//# sourceMappingURL=404.js.map