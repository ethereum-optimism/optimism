import { groupHeaders } from "./groupHeader";
import { ensureEndingSlash, ensureExt, isExternal, normalize, resolvePath, } from "./path";
export const groupSidebarHeaders = groupHeaders;
const resolveSidebarHeaders = (page) => {
    const headers = page.headers ? groupSidebarHeaders(page.headers) : [];
    return [
        {
            type: "group",
            collapsable: false,
            title: page.title,
            icon: page.frontmatter.icon,
            path: "",
            children: headers.map((header) => (Object.assign(Object.assign({}, header), { type: "header", basePath: page.path, path: `${page.path}#${header.slug}`, children: header.children }))),
        },
    ];
};
const findMatchingSidebarConfig = (regularPath, config) => {
    // return directly as array-type config is the moest simple config
    if (Array.isArray(config))
        return {
            base: "/",
            config,
        };
    // find matching config
    for (const base in config)
        if (ensureEndingSlash(regularPath).startsWith(encodeURI(base)))
            return {
                base,
                config: config[base],
            };
    console.warn(`${regularPath} do not have valid sidebar config`);
    return false;
};
/** sidebarConfig merged with pageObject */
export const resolvePageforSidebar = (pages, path) => {
    // if it is external link
    if (isExternal(path))
        return {
            type: "external",
            path,
        };
    const realPath = normalize(path);
    // find matches in all pages
    for (const page of pages)
        if (normalize(page.regularPath) === realPath)
            // return sidebarConfig merged with pageObject
            return Object.assign(Object.assign({}, page), { type: "page", path: ensureExt(page.path) });
    console.error(`Sidebar: "${realPath}" has no matching page`);
    return { type: "error", path: realPath };
};
const resolve = (prefix, path, base) => resolvePath(`${prefix}${path}`, base);
/**
 * @param sidebarConfigItem config item being resolved
 * @param pages pages Object
 * @param base sidebar base
 */
const resolveSidebarItem = (sidebarConfigItem, pages, base, prefix = "") => {
    // resolve and return directly
    if (typeof sidebarConfigItem === "string")
        return resolvePageforSidebar(pages, resolve(prefix, sidebarConfigItem, base));
    // custom title with format `['path', 'customTitle']`
    if (Array.isArray(sidebarConfigItem))
        return Object.assign(resolvePageforSidebar(pages, resolve(prefix, sidebarConfigItem[0], base)), { title: sidebarConfigItem[1] });
    const children = sidebarConfigItem.children || [];
    // item do not have children
    if (children.length === 0 && sidebarConfigItem.path)
        // cover title
        return Object.assign(resolvePageforSidebar(pages, resolve(prefix, sidebarConfigItem.path, base)), { title: sidebarConfigItem.title });
    //  resolve children recursively then return
    return Object.assign(Object.assign({}, sidebarConfigItem), { type: "group", path: sidebarConfigItem.path
            ? resolve(prefix, sidebarConfigItem.path, base)
            : "", children: children.map((child) => resolveSidebarItem(child, pages, base, `${prefix}${sidebarConfigItem.prefix || ""}`)), collapsable: sidebarConfigItem.collapsable !== false });
};
export const getSidebarItems = (page, site, localePath) => {
    const { themeConfig, pages } = site;
    const localeConfig = localePath && themeConfig.locales
        ? themeConfig.locales[localePath] || themeConfig
        : themeConfig;
    const sidebarConfig = localeConfig.sidebar || themeConfig.sidebar;
    // auto generate sidebar through headings
    if (page.frontmatter.sidebar === "auto" || sidebarConfig === "auto")
        return resolveSidebarHeaders(page);
    // sidebar is disabled
    if (!sidebarConfig)
        return [];
    const result = findMatchingSidebarConfig(page.regularPath, sidebarConfig);
    return result
        ? result.config.map((item) => resolveSidebarItem(item, pages, result.base))
        : [];
};
//# sourceMappingURL=sidebar.js.map