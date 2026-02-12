import { SidebarItem } from "vocs";

export const opRethCliSidebar: SidebarItem = {
    text: "op-reth",
    link: "/op-reth/cli/op-reth",
    collapsed: false,
    items: [
        {
            text: "op-reth node",
            link: "/op-reth/cli/op-reth/node"
        },
        {
            text: "op-reth init",
            link: "/op-reth/cli/op-reth/init"
        },
        {
            text: "op-reth init-state",
            link: "/op-reth/cli/op-reth/init-state"
        },
        {
            text: "op-reth import-op",
            link: "/op-reth/cli/op-reth/import-op"
        },
        {
            text: "op-reth import-receipts-op",
            link: "/op-reth/cli/op-reth/import-receipts-op"
        },
        {
            text: "op-reth dump-genesis",
            link: "/op-reth/cli/op-reth/dump-genesis"
        },
        {
            text: "op-reth db",
            link: "/op-reth/cli/op-reth/db",
            collapsed: true,
            items: [
                {
                    text: "op-reth db stats",
                    link: "/op-reth/cli/op-reth/db/stats"
                },
                {
                    text: "op-reth db list",
                    link: "/op-reth/cli/op-reth/db/list"
                },
                {
                    text: "op-reth db checksum",
                    link: "/op-reth/cli/op-reth/db/checksum",
                    collapsed: true,
                    items: [
                        {
                            text: "op-reth db checksum mdbx",
                            link: "/op-reth/cli/op-reth/db/checksum/mdbx"
                        },
                        {
                            text: "op-reth db checksum static-file",
                            link: "/op-reth/cli/op-reth/db/checksum/static-file"
                        }
                    ]
                },
                {
                    text: "op-reth db diff",
                    link: "/op-reth/cli/op-reth/db/diff"
                },
                {
                    text: "op-reth db get",
                    link: "/op-reth/cli/op-reth/db/get",
                    collapsed: true,
                    items: [
                        {
                            text: "op-reth db get mdbx",
                            link: "/op-reth/cli/op-reth/db/get/mdbx"
                        },
                        {
                            text: "op-reth db get static-file",
                            link: "/op-reth/cli/op-reth/db/get/static-file"
                        }
                    ]
                },
                {
                    text: "op-reth db drop",
                    link: "/op-reth/cli/op-reth/db/drop"
                },
                {
                    text: "op-reth db clear",
                    link: "/op-reth/cli/op-reth/db/clear",
                    collapsed: true,
                    items: [
                        {
                            text: "op-reth db clear mdbx",
                            link: "/op-reth/cli/op-reth/db/clear/mdbx"
                        },
                        {
                            text: "op-reth db clear static-file",
                            link: "/op-reth/cli/op-reth/db/clear/static-file"
                        }
                    ]
                },
                {
                    text: "op-reth db repair-trie",
                    link: "/op-reth/cli/op-reth/db/repair-trie"
                },
                {
                    text: "op-reth db static-file-header",
                    link: "/op-reth/cli/op-reth/db/static-file-header",
                    collapsed: true,
                    items: [
                        {
                            text: "op-reth db static-file-header block",
                            link: "/op-reth/cli/op-reth/db/static-file-header/block"
                        },
                        {
                            text: "op-reth db static-file-header path",
                            link: "/op-reth/cli/op-reth/db/static-file-header/path"
                        }
                    ]
                },
                {
                    text: "op-reth db version",
                    link: "/op-reth/cli/op-reth/db/version"
                },
                {
                    text: "op-reth db path",
                    link: "/op-reth/cli/op-reth/db/path"
                },
                {
                    text: "op-reth db settings",
                    link: "/op-reth/cli/op-reth/db/settings",
                    collapsed: true,
                    items: [
                        {
                            text: "op-reth db settings get",
                            link: "/op-reth/cli/op-reth/db/settings/get"
                        },
                        {
                            text: "op-reth db settings set",
                            link: "/op-reth/cli/op-reth/db/settings/set",
                            collapsed: true,
                            items: [
                                {
                                    text: "op-reth db settings set receipts",
                                    link: "/op-reth/cli/op-reth/db/settings/set/receipts"
                                },
                                {
                                    text: "op-reth db settings set transaction_senders",
                                    link: "/op-reth/cli/op-reth/db/settings/set/transaction_senders"
                                },
                                {
                                    text: "op-reth db settings set account_changesets",
                                    link: "/op-reth/cli/op-reth/db/settings/set/account_changesets"
                                },
                                {
                                    text: "op-reth db settings set storages_history",
                                    link: "/op-reth/cli/op-reth/db/settings/set/storages_history"
                                },
                                {
                                    text: "op-reth db settings set transaction_hash_numbers",
                                    link: "/op-reth/cli/op-reth/db/settings/set/transaction_hash_numbers"
                                },
                                {
                                    text: "op-reth db settings set account_history",
                                    link: "/op-reth/cli/op-reth/db/settings/set/account_history"
                                },
                                {
                                    text: "op-reth db settings set storage_changesets",
                                    link: "/op-reth/cli/op-reth/db/settings/set/storage_changesets"
                                }
                            ]
                        }
                    ]
                },
                {
                    text: "op-reth db account-storage",
                    link: "/op-reth/cli/op-reth/db/account-storage"
                }
            ]
        },
        {
            text: "op-reth stage",
            link: "/op-reth/cli/op-reth/stage",
            collapsed: true,
            items: [
                {
                    text: "op-reth stage run",
                    link: "/op-reth/cli/op-reth/stage/run"
                },
                {
                    text: "op-reth stage drop",
                    link: "/op-reth/cli/op-reth/stage/drop"
                },
                {
                    text: "op-reth stage dump",
                    link: "/op-reth/cli/op-reth/stage/dump",
                    collapsed: true,
                    items: [
                        {
                            text: "op-reth stage dump execution",
                            link: "/op-reth/cli/op-reth/stage/dump/execution"
                        },
                        {
                            text: "op-reth stage dump storage-hashing",
                            link: "/op-reth/cli/op-reth/stage/dump/storage-hashing"
                        },
                        {
                            text: "op-reth stage dump account-hashing",
                            link: "/op-reth/cli/op-reth/stage/dump/account-hashing"
                        },
                        {
                            text: "op-reth stage dump merkle",
                            link: "/op-reth/cli/op-reth/stage/dump/merkle"
                        }
                    ]
                },
                {
                    text: "op-reth stage unwind",
                    link: "/op-reth/cli/op-reth/stage/unwind",
                    collapsed: true,
                    items: [
                        {
                            text: "op-reth stage unwind to-block",
                            link: "/op-reth/cli/op-reth/stage/unwind/to-block"
                        },
                        {
                            text: "op-reth stage unwind num-blocks",
                            link: "/op-reth/cli/op-reth/stage/unwind/num-blocks"
                        }
                    ]
                }
            ]
        },
        {
            text: "op-reth p2p",
            link: "/op-reth/cli/op-reth/p2p",
            collapsed: true,
            items: [
                {
                    text: "op-reth p2p header",
                    link: "/op-reth/cli/op-reth/p2p/header"
                },
                {
                    text: "op-reth p2p body",
                    link: "/op-reth/cli/op-reth/p2p/body"
                },
                {
                    text: "op-reth p2p rlpx",
                    link: "/op-reth/cli/op-reth/p2p/rlpx",
                    collapsed: true,
                    items: [
                        {
                            text: "op-reth p2p rlpx ping",
                            link: "/op-reth/cli/op-reth/p2p/rlpx/ping"
                        }
                    ]
                },
                {
                    text: "op-reth p2p bootnode",
                    link: "/op-reth/cli/op-reth/p2p/bootnode"
                }
            ]
        },
        {
            text: "op-reth config",
            link: "/op-reth/cli/op-reth/config"
        },
        {
            text: "op-reth prune",
            link: "/op-reth/cli/op-reth/prune"
        },
        {
            text: "op-reth re-execute",
            link: "/op-reth/cli/op-reth/re-execute"
        }
    ]
};
