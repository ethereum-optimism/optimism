import Vue from "vue";
import Anchor from "@theme/components/Anchor.vue";
import Comment from "@Comment";
import MyTransition from "@theme/components/MyTransition.vue";
import PageInfo from "@mr-hope/vuepress-plugin-comment/lib/client/PageInfo.vue";
import PageMeta from "@theme/components/PageMeta.vue";
import PageNav from "@theme/components/PageNav.vue";
export default Vue.extend({
    name: "Page",
    components: {
        Anchor,
        Comment,
        MyTransition,
        PageInfo,
        PageMeta,
        PageNav,
    },
    props: {
        sidebarItems: {
            type: Array,
            default: () => [],
        },
        headers: {
            type: Array,
            default: () => [],
        },
    },
});
//# sourceMappingURL=Page.js.map
