import Anchor from "@theme/components/Anchor.vue";
import MyTransition from "@theme/components/MyTransition.vue";
import { timelineMixin } from "@theme/mixins/timeline";
import { getDefaultLocale } from "@mr-hope/vuepress-shared";
export default timelineMixin.extend({
    name: "Timeline",
    components: { Anchor, MyTransition },
    computed: {
        hint() {
            return ((this.$themeConfig.blog && this.$themeConfig.blog.timeline) ||
                // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
                this.$themeLocaleConfig.blog.timelineText ||
                getDefaultLocale().blog.timelineText);
        },
        anchorConfig() {
            return this.$timeline.map((item) => ({
                title: item.year.toString(),
                level: 2,
                slug: item.year.toString(),
            }));
        },
    },
    methods: {
        navigate(url) {
            void this.$router.push(url);
        },
    },
});
//# sourceMappingURL=Timeline.js.map