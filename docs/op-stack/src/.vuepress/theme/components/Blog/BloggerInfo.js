import { getDefaultLocale } from "@mr-hope/vuepress-shared";
import MediaLinks from "@theme/components/MediaLinks.vue";
import { timelineMixin } from "@theme/mixins/timeline";
import { filterArticle } from "@theme/utils/article";
import { navigate } from "@theme/utils/navigate";
export default timelineMixin.extend({
    name: "BloggerInfo",
    components: { MediaLinks },
    computed: {
        blogConfig() {
            return this.$themeConfig.blog || {};
        },
        bloggerName() {
            return (this.blogConfig.name ||
                this.$themeConfig.author ||
                this.$site.title ||
                "");
        },
        bloggerAvatar() {
            return this.blogConfig.avatar || this.$themeConfig.logo || "";
        },
        hasIntro() {
            return Boolean(this.blogConfig.intro);
        },
        hintAttr() {
            return this.hasIntro ? "aria-label" : "";
        },
        i18n() {
            return this.$themeLocaleConfig.blog || getDefaultLocale().blog;
        },
        articleNumber() {
            return filterArticle(this.$site.pages).length;
        },
    },
    methods: {
        navigate(url) {
            navigate(url, this.$router, this.$route);
        },
        jumpIntro() {
            if (this.hasIntro)
                navigate(this.blogConfig.intro, this.$router, this.$route);
        },
    },
});
//# sourceMappingURL=BloggerInfo.js.map