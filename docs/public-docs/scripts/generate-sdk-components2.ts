import { Project, SyntaxKind } from "ts-morph";
import * as fs from "fs";
import * as path from "path";

export interface MethodDoc {
  name: string;
  description: string;
  params: Array<{ name: string; type: string; description: string }>;
  returns: string;
  throws?: string;
  signature: string;
  lineNumber: number;
  sourcePath: string;
}

export interface TypeInfo {
  properties: Array<{ name: string; type: string }>;
}

export interface PropertyDoc {
  name: string;
  type: string;
  description: string;
}

export interface ComponentConfig {
  className: string;
  sourcePath: string;
}

export interface GenerationOptions {
  sdkPath: string;
  outputDir: string;
  gitRef: string;
  githubUrlBase: string;
  sdkName: string;
  sdkPackageName: string;
}

export interface ProcessComponentParams extends GenerationOptions {
  component: ComponentConfig;
  project: Project;
}

/**
 * Strip import paths from type names for cleaner documentation output.
 * Example: `import("/path/to/file").LendMarketId` -> `LendMarketId`
 */
function cleanTypeText(typeText: string): string {
  const importPattern = /import\([^)]+\)\.(\w+)/g;
  return typeText.replace(importPattern, "$1");
}

/**
 * Extract type information from a TypeScript type
 */
export function getTypeInfo(type: any, project: Project): TypeInfo | null {
  try {
    const typeText = type.getText();
    const cleanTypeName = typeText.split("<")[0].trim();

    for (const sourceFile of project.getSourceFiles()) {
      const typeAlias = sourceFile.getTypeAlias(cleanTypeName);
      if (typeAlias) {
        const typeNode = typeAlias.getTypeNode();
        if (typeNode && typeNode.getKindName() === "TypeLiteral") {
          return extractPropertiesFromTypeLiteral(typeNode);
        }
      }

      const interfaceDecl = sourceFile.getInterface(cleanTypeName);
      if (interfaceDecl) {
        return extractPropertiesFromInterface(interfaceDecl);
      }
    }

    return null;
  } catch (error) {
    return null;
  }
}

/**
 * Extract properties from a type literal node
 */
function extractPropertiesFromTypeLiteral(typeNode: any): TypeInfo {
  const members = typeNode.getMembers();

  const properties = members
    .filter((member: any) => member.getKindName() === "PropertySignature")
    .map((member: any) => ({
      name: member.getName(),
      type: cleanTypeText(member.getType().getText()),
    }));

  return { properties };
}

/**
 * Extract properties from an interface declaration
 */
function extractPropertiesFromInterface(interfaceDecl: any): TypeInfo {
  const properties = interfaceDecl.getProperties().map((prop: any) => ({
    name: prop.getName(),
    type: cleanTypeText(prop.getType().getText()),
  }));

  return { properties };
}

/**
 * Build a map of parameter types for a method
 */
function buildParamTypeMap(method: any, project: Project): Map<string, any> {
  const paramTypeMap = new Map<string, any>();

  method.getParameters().forEach((param: any) => {
    const paramType = param.getType();
    paramTypeMap.set(param.getName(), paramType);

    const typeInfo = getTypeInfo(paramType, project);
    if (typeInfo) {
      paramTypeMap.set(`${param.getName()}:typeInfo`, typeInfo);
    }
  });

  return paramTypeMap;
}

/**
 * Normalize JSDoc comment text from string or structured array format.
 */
function extractCommentText(commentText: any): string {
  if (typeof commentText === "string") {
    return commentText;
  }
  if (Array.isArray(commentText)) {
    return commentText
      .map((part) => (typeof part === "string" ? part : part.text))
      .join(" ");
  }
  return "";
}

/**
 * Parse nested JSDoc parameter names and look up type information.
 * Example: `params.signer` -> baseName: `params`, propName: `signer`
 */
function processNestedParameter(
  name: string,
  description: string,
  paramTypeMap: Map<string, any>
): { name: string; type: string; description: string } {
  const parts = name.split(".");
  const baseName = parts[0];
  const propName = parts.slice(1).join(".");

  const typeInfo = paramTypeMap.get(`${baseName}:typeInfo`) as
    | TypeInfo
    | undefined;

  if (typeInfo?.properties) {
    const prop = typeInfo.properties.find((p) => p.name === propName);
    if (prop) {
      return {
        name,
        type: prop.type,
        description: description.trim(),
      };
    }
  }

  return {
    name,
    type: "",
    description: description.trim(),
  };
}

/**
 * Process a top-level parameter
 */
function processTopLevelParameter(
  name: string,
  description: string,
  paramTypeMap: Map<string, any>
): { name: string; type: string; description: string } {
  const paramType = paramTypeMap.get(name);

  if (!paramType) {
    return {
      name,
      type: "",
      description: description.trim(),
    };
  }

  const paramTypeText = cleanTypeText(paramType.getText());

  return {
    name,
    type: paramTypeText,
    description: description.trim(),
  };
}

/**
 * Extract method documentation from a class
 */
export function extractMethodDocs(
  classDeclaration: any,
  sourcePath: string,
  project: Project
): MethodDoc[] {
  return classDeclaration
    .getMethods()
    .filter((method: any) => {
      const scope = method.getScope();
      return (
        scope !== "protected" && scope !== "private" && method.getJsDocs()[0]
      );
    })
    .map((method: any) => {
      const jsDoc = method.getJsDocs()[0];
      const tags = jsDoc.getTags();
      const description = jsDoc.getDescription().trim();
      const paramTags = tags.filter((tag: any) => tag.getTagName() === "param");
      const paramTypeMap = buildParamTypeMap(method, project);

      const params = paramTags.map((tag: any) => {
        const commentText = tag.getComment();
        const text = extractCommentText(commentText);
        const name = tag.compilerNode.name?.getText() || "";

        return name.includes(".")
          ? processNestedParameter(name, text, paramTypeMap)
          : processTopLevelParameter(name, text, paramTypeMap);
      });

      const returnsTag = tags.find(
        (tag: any) => tag.getTagName() === "returns"
      );
      const throwsTag = tags.find((tag: any) => tag.getTagName() === "throws");

      return {
        name: method.getName(),
        description,
        params,
        returns: returnsTag?.getCommentText() || "",
        throws: throwsTag?.getCommentText(),
        signature: method.getText(),
        lineNumber: method.getStartLineNumber(),
        sourcePath,
      };
    });
}

/**
 * Extract property documentation from a class
 */
export function extractPropertyDocs(classDeclaration: any): PropertyDoc[] {
  return classDeclaration
    .getProperties()
    .filter((property: any) => {
      const scope = property.getScope();
      return (
        scope !== "protected" && scope !== "private" && property.getJsDocs()[0]
      );
    })
    .map((property: any) => ({
      name: property.getName(),
      type: cleanTypeText(property.getType().getText()),
      description: property.getJsDocs()[0].getDescription().trim(),
    }));
}

/**
 * Generate namespace section of MDX
 */
function generateNamespacesSection(namespaces: PropertyDoc[]): string {
  if (namespaces.length === 0) return "";

  const rows = namespaces
    .map(
      (property) =>
        `| \`${property.name}\` | \`${property.type}\` | ${property.description} |`
    )
    .join("\n");

  return `### Namespaces\n\n| Namespace | Type | Description |\n|-----------|------|-------------|\n${rows}\n\n`;
}

/**
 * Generate properties section of MDX
 */
function generatePropertiesSection(properties: PropertyDoc[]): string {
  if (properties.length === 0) return "";

  const rows = properties
    .map(
      (property) =>
        `| \`${property.name}\` | \`${property.type}\` | ${property.description} |`
    )
    .join("\n");

  return `### Properties\n\n| Property | Type | Description |\n|----------|------|-------------|\n${rows}\n\n`;
}

/**
 * Generate methods table section of MDX
 */
function generateMethodsTableSection(methods: MethodDoc[]): string {
  if (methods.length === 0) return "";

  const rows = methods
    .map((method) => {
      const methodLink = `#${method.name.toLowerCase()}`;
      return `| **[${method.name}()](${methodLink})** | ${
        method.description || ""
      } |`;
    })
    .join("\n");

  return `### Methods\n\n| Function | Description |\n|----------|-------------|\n${rows}\n\n`;
}

/**
 * Generate parameters section for a method
 */
function generateParametersSection(params: MethodDoc["params"]): string {
  if (params.length === 0) return "";

  const rows = params
    .map((param) => {
      // Ensure multiline descriptions remain in the same table cell
      // Before: "- Optional chain IDs.\nIf not provided..."
      // After: "Optional chain IDs. If not provided..."
      const cleanDescription = param.description
        .replace(/^\s*-\s*/, "")
        .replace(/\n/g, " ")
        .replace(/\s+/g, " ")
        .trim();
      const typeCell = param.type ? `\`${param.type}\`` : "";
      return `| \`${param.name}\` | ${typeCell} | ${cleanDescription} |`;
    })
    .join("\n");

  return `| Parameter | Type | Description |\n|-----------|------|-------------|\n${rows}\n\n`;
}

/**
 * Generate method details section of MDX
 */
function generateMethodDetailsSection(
  methods: MethodDoc[],
  githubUrlBase: string
): string {
  return methods
    .map((method) => {
      const githubUrl = `${githubUrlBase}/${method.sourcePath}#L${method.lineNumber}`;
      let mdx = `#### \`${method.name}()\`\n\n`;

      if (method.description) {
        mdx += `${method.description}\n\n`;
      }

      mdx += generateParametersSection(method.params);

      if (method.returns) {
        mdx += `**Returns:** ${method.returns}\n\n`;
      }

      if (method.throws) {
        mdx += `**Throws:** ${method.throws}\n\n`;
      }

      mdx += `<sub>[<Icon icon="github" /> Source ↗](${githubUrl})</sub>\n\n`;
      mdx += `---\n\n`;

      return mdx;
    })
    .join("");
}

/**
 * Generate complete MDX component documentation
 */
export function generateComponentMDX(
  className: string,
  classDescription: string,
  properties: PropertyDoc[],
  methods: MethodDoc[],
  githubUrlBase: string,
  sdkName: string,
  sdkPackageName: string
): string {
  let mdx = `{/*
  ⚠️ WARNING: DO NOT EDIT THIS FILE DIRECTLY ⚠️

  This file is auto-generated from the ${sdkName} source code.

  To update this documentation:
  1. Bump the SDK version in package.json: pnpm add ${sdkPackageName}@latest
  2. Run the generation script: pnpm prebuild

  Any manual edits will be overwritten on the next generation.
*/}

## ${className}

`;

  if (classDescription) {
    mdx += `${classDescription}\n\n`;
  }

  const namespaces = properties.filter((p) => p.type.includes("Namespace"));
  const regularProperties = properties.filter(
    (p) => !p.type.includes("Namespace")
  );

  mdx += generateNamespacesSection(namespaces);
  mdx += generatePropertiesSection(regularProperties);
  mdx += generateMethodsTableSection(methods);
  mdx += generateMethodDetailsSection(methods, githubUrlBase);

  return mdx;
}

/**
 * Convert PascalCase class name to kebab-case filename.
 * Example: `WalletNamespace` -> `wallet-namespace`
 */
export function classNameToFileName(className: string): string {
  return className
    .replace(/([A-Z])/g, "-$1")
    .toLowerCase()
    .slice(1);
}

/**
 * Initialize TypeScript project with SDK source files
 */
export function initializeProject(sdkPath: string): Project {
  const project = new Project({
    compilerOptions: {
      target: 99, // ESNext
      module: 99, // ESNext
    },
  });

  project.addSourceFilesAtPaths(`${sdkPath}/src/**/*.ts`);
  return project;
}

/**
 * Validate and resolve SDK path
 */
export function resolveSdkPath(sdkPath: string): string {
  if (!fs.existsSync(sdkPath)) {
    console.error("SDK not found at:", sdkPath);
    console.error("Make sure the SDK is installed or the path is correct");
    process.exit(1);
  }
  return sdkPath;
}

/**
 * Get git reference for GitHub links
 */
export function getGitRef(sdkPath: string): string {
  const packageJsonPath = path.join(sdkPath, "package.json");
  const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, "utf8"));
  if (!packageJson.version || typeof packageJson.version !== "string") {
    throw new Error("Invalid package.json: missing or invalid version field");
  }
  return `@eth-optimism/actions-sdk@${packageJson.version}`;
}

/**
 * Ensure output directory exists
 */
export function ensureOutputDirectory(outputDir: string): void {
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }
}

/**
 * Process a single component and generate documentation
 */
export function processComponent(params: ProcessComponentParams): void {
  const {
    component,
    project,
    sdkPath,
    outputDir,
    githubUrlBase,
    sdkName,
    sdkPackageName,
  } = params;
  const { className, sourcePath } = component;

  const sourceFilePath = path.join(sdkPath, sourcePath);

  if (!fs.existsSync(sourceFilePath)) {
    console.error(`${sourcePath} not found at: ${sourceFilePath}`);
    return;
  }

  const sourceFile = project.addSourceFileAtPath(sourceFilePath);
  const classDeclaration = sourceFile.getClass(className);

  if (!classDeclaration) {
    console.error(`${className} class not found in ${sourcePath}`);
    return;
  }

  const classJsDoc = classDeclaration.getJsDocs()[0];
  const classDescription = classJsDoc ? classJsDoc.getDescription().trim() : "";

  const properties = extractPropertyDocs(classDeclaration);
  const methods = extractMethodDocs(classDeclaration, sourcePath, project);

  const componentFileName = classNameToFileName(className);
  const componentContent = generateComponentMDX(
    className,
    classDescription,
    properties,
    methods,
    githubUrlBase,
    sdkName,
    sdkPackageName
  );

  const outputPath = path.join(outputDir, `${componentFileName}.mdx`);
  fs.writeFileSync(outputPath, componentContent);

  console.log(`  <${className} />`);
}
