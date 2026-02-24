import { SidebarItem } from "vocs";

export const opAlloySidebar: SidebarItem[] = [
    {
        text: "Introduction",
        items: [
            {
                text: "Overview",
                link: "/op-alloy/intro"
            },
            {
                text: "Getting Started",
                link: "/op-alloy/starting"
            }
        ]
    },
    {
        text: "Building",
        items: [
            {
                text: "Overview",
                link: "/op-alloy/building"
            },
            {
                text: "Consensus",
                link: "/op-alloy/building/consensus"
            },
            {
                text: "Engine RPC Types",
                link: "/op-alloy/building/engine"
            }
        ]
    },
    {
        text: "Reference",
        items: [
            {
                text: "Glossary",
                link: "/op-alloy/glossary"
            }
        ]
    }
];
