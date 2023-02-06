import { getDefaultLocale } from "@mr-hope/vuepress-shared";
import ArticleIcon from "@theme/icons/ArticleIcon.vue";
import CategoryIcon from "@mr-hope/vuepress-plugin-comment/lib/client/icons/CategoryIcon.vue";
import TagIcon from "@mr-hope/vuepress-plugin-comment/lib/client/icons/TagIcon.vue";
import TimeIcon from "@mr-hope/vuepress-plugin-comment/lib/client/icons/TimeIcon.vue";
import ArticleList from "@theme/components/Blog/ArticleList.vue";
import CategoryList from "@theme/components/Blog/CategoryList.vue";
import MyTransition from "@theme/components/MyTransition.vue";
import TagList from "@theme/components/Blog/TagList.vue";
import Timeline from "@theme/components/Blog/Timeline.vue";
import TimelineList from "@theme/components/Blog/TimelineList.vue";
import { filterArticle } from "@theme/utils/article";
import { starMixin } from "@theme/mixins/star";
export default starMixin.extend({
    name: "BlogInfo",
    components: {
        ArticleIcon,
        ArticleList,
        CategoryIcon,
        CategoryList,
        MyTransition,
        TagIcon,
        TagList,
        TimeIcon,
        Timeline,
        TimelineList,
    },
    data: () => ({
        active: "category",
    }),
    computed: {
        i18n() {
            return this.$themeLocaleConfig.blog || getDefaultLocale().blog;
        },
        articleNumber() {
            return filterArticle(this.$site.pages).length;
        },
    },
    methods: {
        setActive(name) {
            this.active = name;
        },
        navigate(path) {
            if (this.$route.path !== path)
                void this.$router.push(path);
        },
    },
});
//# sourceMappingURL=BlogInfoList.js.map