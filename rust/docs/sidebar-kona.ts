import { SidebarItem } from "vocs";

export const konaSidebar: SidebarItem[] = [
  {
    text: "Introduction",
    items: [
      { text: "Overview", link: "/kona/intro/overview" },
      { text: "Why Kona?", link: "/kona/intro/why" },
      { text: "Contributing", link: "/kona/intro/contributing" },
      { text: "Kona Lore", link: "/kona/intro/lore" }
    ]
  },
  {
    text: "Kona for Node Operators",
    items: [
      { text: "System Requirements", link: "/kona/node/requirements" },
      {
          text: "Installation",
          collapsed: true,
          items: [
              {
                  text: "Prerequisites",
                  link: "/kona/node/install/overview"
              },
              {
                  text: "Pre-Built Binaries",
                  link: "/kona/node/install/binaries"
              },
              {
                  text: "Docker",
                  link: "/kona/node/install/docker"
              },
              {
                  text: "Build from Source",
                  link: "/kona/node/install/source"
              }
          ]
      },
      {
          text: "Run a Node",
          items: [
              {
                  text: "Overview",
                  link: "/kona/node/run/overview",
              },
              {
                  text: "Binary",
                  link: "/kona/node/run/binary",
              },
              {
                  text: "Docker",
                  link: "/kona/node/run/docker",
              },
              {
                  text: "How it Works",
                  link: "/kona/node/run/mechanics",
              }
          ]
      },
      {
          text: "JSON-RPC Reference",
          items: [
              {
                  text: "Overview",
                  link: "/kona/node/rpc/overview",
              },
              {
                  text: "p2p",
                  link: "/kona/node/rpc/p2p",
              },
              {
                  text: "rollup",
                  link: "/kona/node/rpc/rollup",
              },
              {
                  text: "admin",
                  link: "/kona/node/rpc/admin",
              }
          ]
      },
      { text: "Configuration", link: "/kona/node/configuration" },
      { text: "Kurtosis Integration", link: "/kona/kurtosis/overview" },
      { text: "Monitoring", link: "/kona/node/monitoring" },
      { text: "Subcommands", link: "/kona/node/subcommands" },
      {
          text: "FAQ",
          link: "/kona/node/faq/overview",
          collapsed: true,
          items: [
              {
                  text: "Ports",
                  link: "/kona/node/faq/ports"
              },
              {
                  text: "Profiling",
                  link: "/kona/node/faq/profiling"
              }
          ]
      }
    ]
  },
  {
    text: "Kona as a Library",
    items: [
      { text: "Overview", link: "/kona/sdk/overview" },
      {
        text: "Node SDK",
        items: [
          { text: "Introduction", link: "/kona/node/design/intro" },
          { text: "Derivation", link: "/kona/node/design/derivation" },
          { text: "Engine", link: "/kona/node/design/engine" },
          { text: "P2P", link: "/kona/node/design/p2p" },
          { text: "Sequencer", link: "/kona/node/design/sequencer" }
        ]
      },
      {
        text: "Proof SDK",
        items: [
          { text: "Introduction", link: "/kona/sdk/proof/intro" },
          { text: "FPVM Backend", link: "/kona/sdk/proof/fpvm-backend" },
          { text: "Custom Backend", link: "/kona/sdk/proof/custom-backend" },
          { text: "kona-executor Extensions", link: "/kona/sdk/proof/exec-ext" }
        ]
      },
      {
        text: "Fault Proof Program Development",
        collapsed: true,
        items: [
          { text: "Introduction", link: "/kona/sdk/fpp-dev/intro" },
          { text: "Environment", link: "/kona/sdk/fpp-dev/env" },
          { text: "Supported Targets", link: "/kona/sdk/fpp-dev/targets" },
          { text: "Prologue", link: "/kona/sdk/fpp-dev/prologue" },
          { text: "Execution", link: "/kona/sdk/fpp-dev/execution" },
          { text: "Epilogue", link: "/kona/sdk/fpp-dev/epilogue" }
        ]
      },
      {
        text: "Protocol Libraries",
        collapsed: true,
        items: [
          { text: "Introduction", link: "/kona/sdk/protocol/intro" },
          { text: "Registry", link: "/kona/sdk/protocol/registry" },
          { text: "Interop", link: "/kona/sdk/protocol/interop" },
          { text: "Hardforks", link: "/kona/sdk/protocol/hardforks" },
          {
            text: "Derivation",
            collapsed: true,
            items: [
              { text: "Introduction", link: "/kona/sdk/protocol/derive/intro" },
              { text: "Custom Providers", link: "/kona/sdk/protocol/derive/providers" },
              { text: "Stage Swapping", link: "/kona/sdk/protocol/derive/stages" },
              { text: "Signaling", link: "/kona/sdk/protocol/derive/signaling" }
            ]
          },
          {
            text: "Genesis",
            collapsed: true,
            items: [
              { text: "Introduction", link: "/kona/sdk/protocol/genesis/intro" },
              { text: "Rollup Config", link: "/kona/sdk/protocol/genesis/rollup-config" },
              { text: "System Config", link: "/kona/sdk/protocol/genesis/system-config" }
            ]
          },
          {
            text: "Protocol",
            collapsed: true,
            items: [
              { text: "Introduction", link: "/kona/sdk/protocol/protocol/intro" },
              { text: "BlockInfo", link: "/kona/sdk/protocol/protocol/block-info" },
              { text: "L2BlockInfo", link: "/kona/sdk/protocol/protocol/l2-block-info" },
              { text: "Frames", link: "/kona/sdk/protocol/protocol/frames" },
              { text: "Channels", link: "/kona/sdk/protocol/protocol/channels" },
              { text: "Batches", link: "/kona/sdk/protocol/protocol/batches" }
            ]
          }
        ]
      },
      {
        text: "Examples",
        collapsed: true,
        items: [
          { text: "Introduction", link: "/kona/sdk/examples/intro" },
          { text: "Load a Rollup Config", link: "/kona/sdk/examples/load-a-rollup-config" },
          { text: "Transform Frames to a Batch", link: "/kona/sdk/examples/frames-to-batch" },
          { text: "Transform a Batch into Frames", link: "/kona/sdk/examples/batch-to-frames" },
          { text: "Create a new L1BlockInfoTx Hardfork Variant", link: "/kona/sdk/examples/new-l1-block-info-tx-hardfork" },
          { text: "Create a new kona-executor test fixture", link: "/kona/sdk/examples/executor-test-fixtures" },
          { text: "Configuring P2P Network Peer Scoring", link: "/kona/sdk/examples/p2p-peer-scoring" },
          { text: "Custom Derivation Pipeline with New Stage", link: "/kona/sdk/examples/custom-derivation-pipeline" },
          { text: "Testing Kona Sequencing with Kurtosis", link: "/kona/sdk/examples/kurtosis-sequencing-test" }
        ]
      }
    ]
  },
  {
    text: "RFC",
    link: "/kona/rfc/active/intro",
    items: [
      {
        text: "Active RFCs",
        items: [ ]
      },
      {
        text: "Archived RFCs",
        collapsed: true,
        items: [
          { text: "Umbrellas", link: "/kona/rfc/archived/umbrellas" },
          { text: "Monorepo", link: "/kona/rfc/archived/monorepo" }
        ]
      }
    ]
  }
];
