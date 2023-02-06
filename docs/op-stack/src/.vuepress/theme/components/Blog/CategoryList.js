import Vue from "vue";
import { capitalize } from "@mr-hope/vuepress-shared";
import { navigate } from "@theme/utils/navigate";
export default Vue.extend({
    name: "CategoryList",
    methods: {
        capitalize,
        clickCategory(path) {
            navigate(path, this.$router, this.$route);
        },
    },
});
//# sourceMappingURL=CategoryList.js.map