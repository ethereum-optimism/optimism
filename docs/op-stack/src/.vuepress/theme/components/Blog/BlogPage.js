import Vue from "vue";
import ArticleList from "@theme/components/Blog/ArticleList.vue";
import ArticleType from "@theme/components/Blog/ArticleType.vue";
import BlogInfo from "@BlogInfo";
import CategoryList from "@theme/components/Blog/CategoryList.vue";
import MyTransition from "@theme/components/MyTransition.vue";
import TagList from "@theme/components/Blog/TagList.vue";
import Timeline from "@theme/components/Blog/Timeline.vue";
import TimelineList from "@theme/components/Blog/TimelineList.vue";
export default Vue.extend({
    name: "BlogPage",
    components: {
        ArticleList,
        ArticleType,
        BlogInfo,
        CategoryList,
        MyTransition,
        TagList,
        Timeline,
        TimelineList,
    },
    computed: {
        showArticles() {
            const { path } = this.$route;
            return !path.includes("/timeline");
        },
        componentName() {
            const pathName = this.$route.path.split("/")[1];
            if (["category", "tag"].includes(pathName))
                return `${pathName}List`;
            else if (pathName === "timeline")
                return pathName;
            return "articleType";
        },
    },
});
//# sourceMappingURL=BlogPage.js.map