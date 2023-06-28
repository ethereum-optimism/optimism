import type { PageComputed, SiteData } from "@mr-hope/vuepress-types";
import type { SidebarHeader } from "./groupHeader";
export type { SidebarHeader } from "./groupHeader";
export interface SidebarHeaderItem extends SidebarHeader {
    type: "header";
    basePath: string;
    path: string;
}
export interface SidebarAutoItem {
    type: "group";
    /** Group title */
    title: string;
    /** Page Icon */
    icon?: string;
    /** Titles in page */
    children: SidebarHeaderItem[];
    collapsable: false;
    path: "";
}
export declare const groupSidebarHeaders: (headers: import("@mr-hope/vuepress-types").PageHeader[]) => SidebarHeader[];
export interface SidebarExternalItem {
    title?: string;
    icon?: string;
    type: "external";
    path: string;
}
export interface SidebarPageItem extends PageComputed {
    type: "page";
    icon?: string;
    path: string;
}
export interface SidebarGroupItem {
    type: "group";
    title: string;
    /** @default true */
    collapsable?: boolean;
    /** @default 1 */
    sidebarDepth?: number;
    icon?: string;
    prefix?: string;
    children: SidebarItem[];
    [props: string]: unknown;
}
export interface SidebarErrorItem {
    type: "error";
    path: string;
}
/** sidebarConfig merged with pageObject */
export declare const resolvePageforSidebar: (pages: PageComputed[], path: string) => SidebarPageItem | SidebarExternalItem | SidebarErrorItem;
export declare type SidebarItem = SidebarAutoItem | SidebarErrorItem | SidebarExternalItem | SidebarGroupItem | SidebarPageItem;
export declare const getSidebarItems: (page: PageComputed, site: SiteData, localePath: string) => SidebarItem[];
