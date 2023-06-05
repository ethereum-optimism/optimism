import type { HopeNavBarConfigItem } from "../types";
export interface NavBarConfigItem extends HopeNavBarConfigItem {
    type: "link" | "links";
    items: NavBarConfigItem[];
}
export declare const getNavLinkItem: (navbarLink: HopeNavBarConfigItem, beforeprefix?: string) => NavBarConfigItem;
