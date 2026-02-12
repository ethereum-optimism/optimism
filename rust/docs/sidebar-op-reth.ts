import { SidebarItem } from "vocs";
import { opRethCliSidebar } from "./sidebar-cli-op-reth";

export const sidebar: SidebarItem[] = [
    {
        text: "Introduction",
        items: [
            {
                text: "Overview",
                link: "/"
            }
        ]
    },
    {
        text: "Running op-reth",
        items: [
            {
                text: "OP Stack",
                link: "/run/opstack"
            },
            {
                text: "FAQ",
                collapsed: true,
                items: [
                    {
                        text: "Sync OP Mainnet",
                        link: "/run/faq/sync-op-mainnet"
                    }
                ]
            }
        ]
    },
    {
        text: "CLI Reference",
        link: "/cli/op-reth",
        collapsed: false,
        items: [
            opRethCliSidebar
        ]
    },
];
