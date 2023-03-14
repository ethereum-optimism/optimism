export const getNavLinkItem = (navbarLink, beforeprefix = "") => {
    var _a;
    const prefix = beforeprefix + (navbarLink.prefix || "");
    const navbarItem = Object.assign({}, navbarLink);
    if (prefix) {
        if (navbarItem.link !== undefined)
            navbarItem.link = prefix + navbarItem.link;
        delete navbarItem.prefix;
    }
    if ((_a = navbarItem.items) === null || _a === void 0 ? void 0 : _a.length)
        Object.assign(navbarItem, {
            type: "links",
            items: navbarItem.items.map((item) => getNavLinkItem(item, prefix)),
        });
    else
        navbarItem.type = "link";
    return navbarItem;
};
//# sourceMappingURL=navbar.js.map