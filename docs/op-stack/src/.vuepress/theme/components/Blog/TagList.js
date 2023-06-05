import Vue from "vue";
import { getDefaultLocale } from "@mr-hope/vuepress-shared";
import { navigate } from "@theme/utils/navigate";
export default Vue.extend({
    name: "TagList",
    computed: {
        tagList() {
            return [
                {
                    name: 
                    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
                    this.$themeLocaleConfig.blog.allText ||
                        getDefaultLocale().blog.allText,
                    path: "/tag/",
                },
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                ...this.$tag.list,
            ];
        },
    },
    methods: {
        isActive(name) {
            return (name ===
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                ((this.$currentTag && this.$currentTag.key) ||
                    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
                    this.$themeLocaleConfig.blog.allText ||
                    getDefaultLocale().blog.allText));
        },
        clickTag(path) {
            navigate(path, this.$router, this.$route);
        },
    },
});
//# sourceMappingURL=TagList.js.map