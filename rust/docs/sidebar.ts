import { SidebarItem } from "vocs";
import { konaSidebar } from "./sidebar-kona";
import { opRethSidebar } from "./sidebar-op-reth";
import { opAlloySidebar } from "./sidebar-op-alloy";

export const sidebar = {
  "/kona/": konaSidebar,
  "/op-reth/": opRethSidebar,
  "/op-alloy/": opAlloySidebar,
} satisfies Record<string, SidebarItem[]>;
