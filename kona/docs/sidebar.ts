import { SidebarItem } from "vocs";

export const sidebar: SidebarItem[] = [
  {
    text: "Introduction",
    items: [
      { text: "Overview", link: "/intro/overview" },
      { text: "Why Kona?", link: "/intro/why" },
      { text: "Contributing", link: "/intro/contributing" },
      { text: "Kona Lore", link: "/intro/lore" }
    ]
  },
  {
    text: "Kona for Node Operators",
    items: [
      { text: "System Requirements", link: "/node/requirements" },
      {
          text: "Installation",
          collapsed: true,
          items: [
              {
                  text: "Prerequisites",
                  link: "/node/install/overview"
              },
              {
                  text: "Pre-Built Binaries",
                  link: "/node/install/binaries"
              },
              {
                  text: "Docker",
                  link: "/node/install/docker"
              },
              {
                  text: "Build from Source",
                  link: "/node/install/source"
              }
          ]
      },
      {
          text: "Run a Node",
          items: [
              {
                  text: "Overview",
                  link: "/node/run/overview",
              },
              {
                  text: "Binary",
                  link: "/node/run/binary",
              },
              {
                  text: "Docker",
                  link: "/node/run/docker",
              },
              {
                  text: "How it Works",
                  link: "/node/run/mechanics",
              }
          ]
      },
      {
          text: "JSON-RPC Reference",
          items: [
              {
                  text: "Overview",
                  link: "/node/rpc/overview",
              },
              {
                  text: "p2p",
                  link: "/node/rpc/p2p",
              },
              {
                  text: "rollup",
                  link: "/node/rpc/rollup",
              },
              {
                  text: "admin",
                  link: "/node/rpc/admin",
              }
          ]
      },
      { text: "Configuration", link: "/node/configuration" },
      { text: "Kurtosis Integration", link: "/kurtosis/overview" },
      { text: "Monitoring", link: "/node/monitoring" },
      { text: "Subcommands", link: "/node/subcommands" },
      {
          text: "FAQ",
          link: "/node/faq/overview",
          collapsed: true,
          items: [
              {
                  text: "Ports",
                  link: "/node/faq/ports"
              },
              {
                  text: "Profiling",
                  link: "/node/faq/profiling"
              }
          ]
      }
    ]
  },
  {
    text: "Kona as a Library",
    items: [
      { text: "Overview", link: "/sdk/overview" },
      {
        text: "Node SDK",
        items: [
          { text: "Introduction", link: "/node/design/intro" },
          { text: "Derivation", link: "/node/design/derivation" },
          { text: "Engine", link: "/node/design/engine" },
          { text: "P2P", link: "/node/design/p2p" },
          { text: "Sequencer", link: "/node/design/sequencer" }
        ]
      },
      {
        text: "Proof SDK",
        items: [
          { text: "Introduction", link: "/sdk/proof/intro" },
          { text: "FPVM Backend", link: "/sdk/proof/fpvm-backend" },
          { text: "Custom Backend", link: "/sdk/proof/custom-backend" },
          { text: "kona-executor Extensions", link: "/sdk/proof/exec-ext" }
        ]
      },
      {
        text: "Fault Proof Program Development",
        collapsed: true,
        items: [
          { text: "Introduction", link: "/sdk/fpp-dev/intro" },
          { text: "Environment", link: "/sdk/fpp-dev/env" },
          { text: "Supported Targets", link: "/sdk/fpp-dev/targets" },
          { text: "Prologue", link: "/sdk/fpp-dev/prologue" },
          { text: "Execution", link: "/sdk/fpp-dev/execution" },
          { text: "Epilogue", link: "/sdk/fpp-dev/epilogue" }
        ]
      },
      {
        text: "Protocol Libraries",
        collapsed: true,
        items: [
          { text: "Introduction", link: "/sdk/protocol/intro" },
          { text: "Registry", link: "/sdk/protocol/registry" },
          { text: "Interop", link: "/sdk/protocol/interop" },
          { text: "Hardforks", link: "/sdk/protocol/hardforks" },
          {
            text: "Derivation",
            collapsed: true,
            items: [
              { text: "Introduction", link: "/sdk/protocol/derive/intro" },
              { text: "Custom Providers", link: "/sdk/protocol/derive/providers" },
              { text: "Stage Swapping", link: "/sdk/protocol/derive/stages" },
              { text: "Signaling", link: "/sdk/protocol/derive/signaling" }
            ]
          },
          {
            text: "Genesis",
            collapsed: true,
            items: [
              { text: "Introduction", link: "/sdk/protocol/genesis/intro" },
              { text: "Rollup Config", link: "/sdk/protocol/genesis/rollup-config" },
              { text: "System Config", link: "/sdk/protocol/genesis/system-config" }
            ]
          },
          {
            text: "Protocol",
            collapsed: true,
            items: [
              { text: "Introduction", link: "/sdk/protocol/protocol/intro" },
              { text: "BlockInfo", link: "/sdk/protocol/protocol/block-info" },
              { text: "L2BlockInfo", link: "/sdk/protocol/protocol/l2-block-info" },
              { text: "Frames", link: "/sdk/protocol/protocol/frames" },
              { text: "Channels", link: "/sdk/protocol/protocol/channels" },
              { text: "Batches", link: "/sdk/protocol/protocol/batches" }
            ]
          }
        ]
      },
      {
        text: "Examples",
        collapsed: true,
        items: [
          { text: "Introduction", link: "/sdk/examples/intro" },
          { text: "Load a Rollup Config", link: "/sdk/examples/load-a-rollup-config" },
          { text: "Transform Frames to a Batch", link: "/sdk/examples/frames-to-batch" },
          { text: "Transform a Batch into Frames", link: "/sdk/examples/batch-to-frames" },
          { text: "Create a new L1BlockInfoTx Hardfork Variant", link: "/sdk/examples/new-l1-block-info-tx-hardfork" },
          { text: "Create a new kona-executor test fixture", link: "/sdk/examples/executor-test-fixtures" },
          { text: "Configuring P2P Network Peer Scoring", link: "/sdk/examples/p2p-peer-scoring" },
          { text: "Custom Derivation Pipeline with New Stage", link: "/sdk/examples/custom-derivation-pipeline" },
          { text: "Testing Kona Sequencing with Kurtosis", link: "/sdk/examples/kurtosis-sequencing-test" }
        ]
      }
    ]
  },
  {
    text: "RFC",
    link: "/rfc/active/intro",
    items: [
      {
        text: "Active RFCs",
        items: [ ]
      },
      {
        text: "Archived RFCs",
        collapsed: true,
        items: [
          { text: "Umbrellas", link: "/rfc/archived/umbrellas" },
          { text: "Monorepo", link: "/rfc/archived/monorepo" }
        ]
      }
    ]
  }
];
