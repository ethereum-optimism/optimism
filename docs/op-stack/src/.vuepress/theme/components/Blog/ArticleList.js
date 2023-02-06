import Vue from "vue";
import ArticleItem from "@theme/components/Blog/ArticleItem.vue";
import EmptyIcon from "@theme/icons/EmptyIcon.vue";
import MyTransition from "@theme/components/MyTransition.vue";
import { filterArticle, sortArticle } from "@theme/utils/article";
import { getPathMatchedKeys } from "@theme/utils/encrypt";
export default Vue.extend({
    name: "ArticleList",
    components: { ArticleItem, EmptyIcon, MyTransition },
    data: () => ({
        currentPage: 1,
        articleList: [],
    }),
    computed: {
        blogConfig() {
            return this.$themeConfig.blog || {};
        },
        articlePerPage() {
            return this.blogConfig.perPage || 10;
        },
        filter() {
            const { path } = this.$route;
            return path.includes("/article")
                ? (page) => page.frontmatter.layout !== "Slide"
                : path.includes("/star")
                    ? (page) => Boolean(page.frontmatter.star || page.frontmatter.sticky)
                    : path.includes("/encrypt")
                        ? (page) => getPathMatchedKeys(this.$themeConfig.encrypt, page.path).length !==
                            0 || Boolean(page.frontmatter.password)
                        : path.includes("/slide")
                            ? (page) => page.frontmatter.layout === "Slide"
                            : undefined;
        },
        $articles() {
            // filter then sort
            return sortArticle(filterArticle(this.$site.pages, this.filter), "sticky");
        },
        /** Articles in this page */
        articles() {
            return this.articleList.slice((this.currentPage - 1) * this.articlePerPage, this.currentPage * this.articlePerPage);
        },
    },
    watch: {
        // update article list when route is changed
        $route(to, from) {
            if (to.path !== from.path) {
                this.articleList = this.getArticleList();
                // reset page to 1
                this.currentPage = 1;
            }
        },
        currentPage() {
            // list top border distance
            const distance = document.querySelector("#article-list").getBoundingClientRect().top + window.scrollY;
            setTimeout(() => {
                window.scrollTo(0, distance);
            }, 100);
        },
    },
    mounted() {
        this.articleList = this.getArticleList();
    },
    methods: {
        getArticleList() {
            try {
                return this.$pagination
                    ? // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                        this.$pagination._matchedPages
                    : this.$articles;
            }
            catch (err) {
                return this.$articles;
            }
        },
    },
});
//# sourceMappingURL=ArticleList.js.map