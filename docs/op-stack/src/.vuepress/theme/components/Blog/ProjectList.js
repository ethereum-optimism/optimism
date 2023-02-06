import Vue from "vue";
import ArticleIcon from "@theme/icons/ArticleIcon.vue";
import BookIcon from "@theme/icons/BookIcon.vue";
import LinkIcon from "@theme/icons/LinkIcon.vue";
import ProjectIcon from "@theme/icons/ProjectIcon.vue";
import { navigate } from "@theme/utils/navigate";
export default Vue.extend({
    name: "ProjectList",
    components: { ArticleIcon, BookIcon, LinkIcon, ProjectIcon },
    methods: {
        navigate(link) {
            navigate(link, this.$router, this.$route);
        },
    },
});
//# sourceMappingURL=ProjectList.js.map