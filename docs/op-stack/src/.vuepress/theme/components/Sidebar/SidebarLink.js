import Vue from "vue";
import { hashRE, isActive } from "@theme/utils/path";
import { groupSidebarHeaders } from "@theme/utils/sidebar";
const renderIcon = (h, icon) => icon
    ? h("i", {
        class: ["iconfont", icon],
    })
    : null;
const renderLink = (h, { icon = "", text, link, level, active }) => h("RouterLink", {
    props: {
        to: link,
        activeClass: "",
        exactActiveClass: "",
    },
    class: {
        active,
        "sidebar-link": true,
        [level ? `heading${level}` : ""]: level && level !== 2,
    },
}, [renderIcon(h, icon), text]);
const renderExternalLink = (h, { path, title = path }) => h("a", {
    attrs: {
        href: path,
        target: "_blank",
        rel: "noopener noreferrer",
    },
    class: { "sidebar-link": true },
}, [title, h("OutboundLink")]);
const renderChildren = (h, { children, path, route, maxDepth, depth = 1 }) => {
    if (!children || depth > maxDepth)
        return null;
    return h("ul", { class: "sidebar-sub-headers" }, children.map((child) => {
        const active = isActive(route, `${path}#${child.slug}`);
        return h("li", { class: "sidebar-sub-header" }, [
            renderLink(h, {
                text: child.title,
                link: `${path}#${child.slug}`,
                level: child.level,
                active,
            }),
            renderChildren(h, {
                children: child.children || false,
                path,
                route,
                maxDepth,
                depth: depth + 1,
            }),
        ]);
    }));
};
export default Vue.extend({
    name: "SidebarLink",
    functional: true,
    props: {
        item: {
            type: Object,
            required: true,
        },
    },
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    render(h, { parent: { $page, $route, $themeConfig, $themeLocaleConfig }, props }) {
        const { item } = props;
        // if the item can not be resolved
        if (item.type === "error")
            return null;
        // external link
        if (item.type === "external")
            return renderExternalLink(h, item);
        /*
         * Use custom active class matching logic
         * Due to edge case of paths ending with / + hash
         */
        const selfActive = isActive($route, item.path);
        /** whether the item is active */
        const active = 
        // if the item is a heading, then one of the children needs to be active
        item.type === "header"
            ? selfActive ||
                (item.children || []).some((child) => isActive($route, `${item.basePath}#${child.slug}`))
            : selfActive;
        const pageMaxDepth = $page.frontmatter.sidebarDepth;
        const localesMaxDepth = $themeLocaleConfig.sidebarDepth;
        const themeMaxDepth = $themeConfig.sidebarDepth;
        const maxDepth = typeof pageMaxDepth === "number"
            ? pageMaxDepth
            : typeof localesMaxDepth === "number"
                ? localesMaxDepth
                : typeof themeMaxDepth === "number"
                    ? themeMaxDepth
                    : 2;
        // the item is a heading
        if (item.type === "header")
            return [
                renderLink(h, {
                    text: item.title || item.path,
                    link: item.path,
                    level: item.level,
                    active,
                }),
                renderChildren(h, {
                    children: item.children || false,
                    path: item.basePath,
                    route: $route,
                    maxDepth,
                }),
            ];
        const displayAllHeaders = $themeLocaleConfig.displayAllHeaders ||
            $themeConfig.displayAllHeaders;
        const link = renderLink(h, {
            icon: $themeConfig.sidebarIcon !== false && item.frontmatter.icon
                ? `${$themeConfig.iconPrefix === ""
                    ? ""
                    : $themeConfig.iconPrefix || "icon-"}${item.frontmatter.icon}`
                : "",
            text: item.title || item.path,
            link: item.path,
            active,
        });
        if ((active || displayAllHeaders) &&
            item.headers &&
            !hashRE.test(item.path)) {
            const children = groupSidebarHeaders(item.headers);
            return [
                link,
                renderChildren(h, {
                    children,
                    path: item.path,
                    route: $route,
                    maxDepth,
                }),
            ];
        }
        return link;
    },
});
//# sourceMappingURL=SidebarLink.js.map