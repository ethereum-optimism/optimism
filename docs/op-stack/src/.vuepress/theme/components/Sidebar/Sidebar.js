import Vue from "vue";
import BlogInfo from "@BlogInfo";
import BloggerInfo from "@BloggerInfo";
import SidebarNavLinks from "@theme/components/Sidebar/SidebarNavLinks.vue";
import SidebarLinks from "@theme/components/Sidebar/SidebarLinks.vue";
export default Vue.extend({
    name: "Sidebar",
    components: {
        BlogInfo,
        BloggerInfo,
        SidebarLinks,
        SidebarNavLinks,
    },
    props: {
        items: { type: Array, required: true },
    },
    computed: {
        blogConfig() {
            return this.$themeConfig.blog || {};
        },
        sidebarDisplay() {
            return this.blogConfig.sidebarDisplay || "none";
        },
    },
});
//# sourceMappingURL=Sidebar.js.map