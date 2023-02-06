import Vue from "vue";
import { filterArticle, sortArticle } from "@theme/utils/article";
export const starMixin = Vue.extend({
    computed: {
        $starArticles() {
            const { pages } = this.$site;
            // filter before sort
            return sortArticle(filterArticle(pages, (page) => Boolean(page.frontmatter.star)), "star");
        },
    },
});
//# sourceMappingURL=star.js.map