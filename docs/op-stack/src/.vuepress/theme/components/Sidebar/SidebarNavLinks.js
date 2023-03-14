import Vue from "vue";
import RepoLink from "@theme/components/Navbar/RepoLink.vue";
import SidebarDropdownLink from "@theme/components/Sidebar/SidebarDropdownLink.vue";
import NavLink from "@theme/components/Navbar/NavLink.vue";
import { getNavLinkItem } from "@theme/utils/navbar";
export default Vue.extend({
    name: "SidebarNavLinks",
    components: {
        RepoLink,
        SidebarDropdownLink,
        NavLink,
    },
    computed: {
        navLinks() {
            const navbar = this.$themeLocaleConfig.nav || this.$themeConfig.nav || [];
            return navbar.map((link) => getNavLinkItem(link));
        },
    },
});
//# sourceMappingURL=SidebarNavLinks.js.map