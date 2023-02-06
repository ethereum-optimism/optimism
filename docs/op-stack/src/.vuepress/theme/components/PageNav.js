import Vue from "vue";
import NextIcon from "@theme/icons/NextIcon.vue";
import PrevIcon from "@theme/icons/PrevIcon.vue";
import { resolvePath } from "@theme/utils/path";
import { resolvePageforSidebar } from "@theme/utils/sidebar";
const getSidebarItems = (items, result) => {
    for (const item of items)
        if (item.type === "group")
            getSidebarItems((item.children || []), result);
        else
            result.push(item);
};
const find = (page, items, offset) => {
    const result = [];
    getSidebarItems(items, result);
    for (let i = 0; i < result.length; i++) {
        const cur = result[i];
        if (cur.type === "page" && cur.path === decodeURIComponent(page.path))
            return result[i + offset];
    }
    return false;
};
const resolvePageLink = (linkType, { themeConfig, page, route, site, sidebarItems }) => {
    const themeLinkConfig = 
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
    themeConfig[`${linkType}Links`];
    const pageLinkConfig = page.frontmatter[linkType];
    if (themeLinkConfig === false || pageLinkConfig === false)
        return false;
    if (typeof pageLinkConfig === "string")
        return resolvePageforSidebar(site.pages, resolvePath(pageLinkConfig, route.path));
    return find(page, sidebarItems, linkType === "prev" ? -1 : 1);
};
export default Vue.extend({
    name: "PageNav",
    components: { NextIcon, PrevIcon },
    props: {
        sidebarItems: {
            type: Array,
            default: () => [],
        },
    },
    computed: {
        prev() {
            return resolvePageLink("prev", {
                sidebarItems: this.sidebarItems,
                themeConfig: this.$themeConfig,
                page: this.$page,
                route: this.$route,
                site: this.$site,
            });
        },
        next() {
            return resolvePageLink("next", {
                sidebarItems: this.sidebarItems,
                themeConfig: this.$themeConfig,
                page: this.$page,
                route: this.$route,
                site: this.$site,
            });
        },
    },
});
//# sourceMappingURL=PageNav.js.map