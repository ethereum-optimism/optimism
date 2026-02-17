import { SidebarItem } from "vocs";
import { opRethCliSidebar } from "./sidebar-cli-op-reth";

export const opRethSidebar: SidebarItem[] = [
    {
        text: "Introduction",
        items: [
            {
                text: "Overview",
                link: "/op-reth/"
            }
        ]
    },
    {
        text: "Running op-reth",
        items: [
            {
                text: "OP Stack",
                link: "/op-reth/run/opstack"
            },
            {
                text: "FAQ",
                collapsed: true,
                items: [
                    {
                        text: "Sync OP Mainnet",
                        link: "/op-reth/run/faq/sync-op-mainnet"
                    }
                ]
            }
        ]
    },
    {
        text: "CLI Reference",
        link: "/op-reth/cli/op-reth",
        collapsed: false,
        items: [
            opRethCliSidebar
        ]
    },
];
