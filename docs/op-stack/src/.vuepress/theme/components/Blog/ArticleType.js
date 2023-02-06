import Vue from "vue";
import { getDefaultLocale } from "@mr-hope/vuepress-shared";
import { navigate } from "@theme/utils/navigate";
export default Vue.extend({
    name: "ArticleType",
    computed: {
        types() {
            const blogI18n = this.$themeLocaleConfig.blog || getDefaultLocale().blog;
            return [
                { text: blogI18n.allText, path: "/article/" },
                { text: blogI18n.star, path: "/star/" },
                { text: blogI18n.slides, path: "/slide/" },
                { text: blogI18n.encrypt, path: "/encrypt/" },
            ];
        },
    },
    methods: {
        navigate(path) {
            navigate(path, this.$router, this.$route);
        },
    },
});
//# sourceMappingURL=ArticleType.js.map