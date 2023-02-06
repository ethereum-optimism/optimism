import MyTransition from "@theme/components/MyTransition.vue";
import TimeIcon from "@mr-hope/vuepress-plugin-comment/lib/client/icons/TimeIcon.vue";
import { timelineMixin } from "@theme/mixins/timeline";
import { getDefaultLocale } from "@mr-hope/vuepress-shared";
export default timelineMixin.extend({
    name: "TimelineList",
    components: { MyTransition, TimeIcon },
    computed: {
        hint() {
            return (
            // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
            this.$themeLocaleConfig.blog.timeline ||
                getDefaultLocale().blog.timeline);
        },
    },
    methods: {
        navigate(url) {
            void this.$router.push(url);
        },
    },
});
//# sourceMappingURL=TimelineList.js.map