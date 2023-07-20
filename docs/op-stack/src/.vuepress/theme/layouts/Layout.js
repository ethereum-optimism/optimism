import Vue from "vue";
import BlogInfo from "@BlogInfo";
import BlogHome from "@BlogHome";
import ContentBottom from "@ContentBottom";
import ContentTop from "@ContentTop";
import NavbarStart from "@NavbarStart";
import NavbarCenter from "@NavbarCenter";
import NavbarEnd from "@NavbarEnd";
import PageBottom from "@PageBottom";
import PageTop from "@PageTop";
import SidebarBottom from "@SidebarBottom";
import SidebarCenter from "@SidebarCenter";
import SidebarTop from "@SidebarTop";
import Common from "@theme/components/Common.vue";
import Home from "@theme/components/Home.vue";
import Page from "@theme/components/Page.vue";
export default Vue.extend({
    name: "Layout",
    components: {
        BlogInfo,
        BlogHome,
        Common,
        ContentBottom,
        ContentTop,
        Home,
        NavbarCenter,
        NavbarEnd,
        NavbarStart,
        Page,
        PageBottom,
        PageTop,
        SidebarBottom,
        SidebarCenter,
        SidebarTop,
    },
});
//# sourceMappingURL=Layout.js.map