import Vue from "vue";
import MediaLinks from "@theme/components/MediaLinks.vue";
export default Vue.extend({
    name: "PageFooter",
    components: { MediaLinks },
    computed: {
        footerConfig() {
            return this.$themeLocaleConfig.footer || this.$themeConfig.footer || {};
        },
        enable() {
            const { copyrightText, footer, medialink } = this.$page.frontmatter;
            return (footer !== false &&
                Boolean(copyrightText || footer || medialink || this.footerConfig.display));
        },
        footerContent() {
            const { footer } = this.$page.frontmatter;
            return footer === false
                ? false
                : typeof footer === "string"
                    ? footer
                    : this.footerConfig.content || "";
        },
        copyright() {
            return this.$frontmatter.copyrightText === false
                ? false
                : this.$frontmatter.copyrightText ||
                    (this.footerConfig.copyright === false
                        ? false
                        : this.footerConfig.copyright ||
                            (this.$themeConfig.author
                                ? `Copyright © ${new Date().getFullYear()} ${this.$themeConfig.author}`
                                : ""));
        },
    },
});
//# sourceMappingURL=PageFooter.js.map