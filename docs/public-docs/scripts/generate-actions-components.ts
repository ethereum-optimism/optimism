import * as path from "path";
import {
  ComponentConfig,
  resolveSdkPath,
  getGitRef,
  ensureOutputDirectory,
  initializeProject,
  processComponent,
} from "./generate-sdk-components.js";

// Define which components to generate
// Key: Component Name, Value: path to source file relative to SDK root
const ACTIONS_COMPONENTS: Record<string, string> = {
  WalletNamespace: "src/wallet/core/namespace/WalletNamespace.ts",
  Wallet: "src/wallet/core/wallets/abstract/Wallet.ts",
  WalletLendNamespace: "src/lend/namespaces/WalletLendNamespace.ts",
};

// SDK metadata
const SDK_NAME = "Actions SDK";
const SDK_PACKAGE_NAME = "@eth-optimism/actions-sdk";

// SDK path configuration
const LOCAL_SDK_PATH = path.join(
  process.cwd(),
  "..",
  "actions",
  "packages",
  "sdk"
);
const NODE_MODULES_SDK_PATH = path.join(
  process.cwd(),
  "node_modules",
  SDK_PACKAGE_NAME
);
const OUTPUT_DIR = path.join(process.cwd(), "snippets", "actions");

async function main() {
  // Parse command line arguments
  const useLocal = process.argv.includes("--local");

  // Determine SDK path based on mode
  const selectedSdkPath = useLocal ? LOCAL_SDK_PATH : NODE_MODULES_SDK_PATH;
  const sdkPath = resolveSdkPath(selectedSdkPath);
  const gitRef = getGitRef(sdkPath);
  const project = initializeProject(sdkPath);
  const githubUrlBase = `https://github.com/ethereum-optimism/actions/blob/${gitRef}/packages/sdk`;

  ensureOutputDirectory(OUTPUT_DIR);

  // Process each component
  const components: ComponentConfig[] = Object.entries(ACTIONS_COMPONENTS).map(
    ([className, sourcePath]) => ({ className, sourcePath })
  );

  console.log("Generating component docs from SDK:");
  components.forEach((component) => {
    processComponent({
      component,
      project,
      sdkPath,
      outputDir: OUTPUT_DIR,
      gitRef,
      githubUrlBase,
      sdkName: SDK_NAME,
      sdkPackageName: SDK_PACKAGE_NAME,
    });
  });
}

main().catch(console.error);
