import * as fs from "fs";
import * as path from "path";
import {
  Project,
  ClassDeclaration,
  MethodDeclaration,
  Scope,
} from "ts-morph";

export interface ComponentConfig {
  className: string;
  sourcePath: string;
}

// ─── SDK path helpers ────────────────────────────────────────────────────────

export function resolveSdkPath(sdkPath: string): string {
  if (!fs.existsSync(sdkPath)) {
    throw new Error(
      `SDK path not found: ${sdkPath}. Make sure dependencies are installed.`
    );
  }
  return sdkPath;
}

export function getGitRef(sdkPath: string): string {
  const pkgPath = path.join(sdkPath, "package.json");
  if (!fs.existsSync(pkgPath)) return "main";
  const pkg = JSON.parse(fs.readFileSync(pkgPath, "utf-8"));
  return `${pkg.name}@${pkg.version}`;
}

export function ensureOutputDirectory(outputDir: string): void {
  fs.mkdirSync(outputDir, { recursive: true });
}

export function initializeProject(sdkPath: string): Project {
  const tsConfigPath = path.join(sdkPath, "tsconfig.json");
  let project: Project;
  if (fs.existsSync(tsConfigPath)) {
    project = new Project({ tsConfigFilePath: tsConfigPath });
  } else {
    // Installed npm package has no tsconfig — configure path aliases manually.
    // @/* maps to src/* (NodeNext module resolution, .js extensions in imports).
    project = new Project({
      compilerOptions: {
        baseUrl: sdkPath,
        paths: { "@/*": ["src/*"] },
        // ModuleResolutionKind.NodeNext = 99
        moduleResolution: 99 as any,
        strict: true,
      },
    });
  }
  // Add all SDK source files so type aliases / interfaces can be resolved
  // across files when documenting nested parameter types.
  project.addSourceFilesAtPaths(`${sdkPath}/src/**/*.ts`);
  return project;
}

// ─── Type resolution helpers ────────────────────────────────────────────────

/**
 * Strip `import("/path/to/file").Foo` → `Foo` for cleaner docs output.
 */
function cleanTypeText(typeText: string): string {
  return typeText.replace(/import\([^)]+\)\.\s*/g, "");
}

/**
 * Get the readable type string for a node, preferring the written alias name
 * (e.g. `LocalAccount`) over the fully-expanded structural type.
 */
function readableType(node: any): string {
  return node.getTypeNode?.()?.getText() ?? cleanTypeText(node.getType().getText());
}

/** Extract property name→type pairs from a type literal's members. */
function propsFromTypeLiteral(typeNode: any): Map<string, string> {
  const props = new Map<string, string>();
  for (const member of typeNode.getMembers()) {
    if (member.getKindName() === "PropertySignature") {
      props.set(member.getName(), readableType(member));
    }
  }
  return props;
}

/** Extract property name→type pairs from an interface declaration. */
function propsFromInterface(iface: any): Map<string, string> {
  const props = new Map<string, string>();
  for (const prop of iface.getProperties()) {
    props.set(prop.getName(), readableType(prop));
  }
  return props;
}

/**
 * Resolve the properties of a named type (type alias or interface) by
 * searching all source files in the project.
 */
function resolveTypeProperties(
  typeName: string,
  project: Project
): Map<string, string> | null {
  const baseName = typeName.split("<")[0].trim(); // strip generics

  for (const sf of project.getSourceFiles()) {
    const alias = sf.getTypeAlias(baseName);
    if (alias) {
      const typeNode = alias.getTypeNode();
      if (typeNode?.getKindName() === "TypeLiteral") return propsFromTypeLiteral(typeNode);
    }

    const iface = sf.getInterface(baseName);
    if (iface) return propsFromInterface(iface);
  }

  return null;
}

// ─── Class / method traversal ────────────────────────────────────────────────

/**
 * Collect all public instance methods visible on a class, walking up the
 * inheritance chain.  Base-class methods come first; overrides in the subclass
 * replace them in-place so the order is stable.
 */
function collectPublicMethods(classDecl: ClassDeclaration): MethodDeclaration[] {
  const result: MethodDeclaration[] = [];
  const nameToIndex = new Map<string, number>();

  // Base class methods first (recursive)
  try {
    const base = classDecl.getBaseClass();
    if (base) {
      for (const m of collectPublicMethods(base)) {
        nameToIndex.set(m.getName(), result.length);
        result.push(m);
      }
    }
  } catch {
    // Base class unresolvable — skip
  }

  // This class's own methods
  for (const method of classDecl.getMethods()) {
    const scope = method.getScope();
    if (scope === Scope.Private || scope === Scope.Protected) continue;

    const name = method.getName();
    if (name.startsWith("#")) continue; // private field syntax

    const existingIdx = nameToIndex.get(name);
    if (existingIdx !== undefined) {
      result[existingIdx] = method; // Replace base version with override
    } else {
      nameToIndex.set(name, result.length);
      result.push(method);
    }
  }

  return result;
}

interface PropertyRow {
  name: string;
  type: string;
  description: string;
}

/** Collect public, JSDoc-documented properties from a class. */
function collectPublicProperties(classDecl: ClassDeclaration): PropertyRow[] {
  const rows: PropertyRow[] = [];
  for (const prop of classDecl.getProperties()) {
    const scope = prop.getScope();
    if (scope === Scope.Private || scope === Scope.Protected) continue;
    const docs = prop.getJsDocs();
    if (docs.length === 0) continue;
    rows.push({
      name: prop.getName(),
      type: readableType(prop),
      description: docs[docs.length - 1].getDescription().trim(),
    });
  }
  return rows;
}

// ─── JSDoc helpers ───────────────────────────────────────────────────────────

function tagComment(tag: any): string {
  const raw = tag.getComment?.();
  if (!raw) return "";
  const text =
    typeof raw === "string"
      ? raw
      : raw
          .map((c: any) =>
            typeof c === "string" ? c : c.getText?.() ?? c.text ?? ""
          )
          .join("");
  // Strip leading "- " separator common in TS JSDoc (@param x - description)
  return text.trim().replace(/^-\s*/, "");
}

/** Collapse newlines to spaces — used for table cells */
function oneLiner(s: string): string {
  return s.replace(/\s*\n\s*/g, " ").trim();
}

interface ParamRow {
  name: string;
  type: string;
  description: string;
}

function extractParams(method: MethodDeclaration, project: Project): ParamRow[] {
  const jsDocParams = new Map<string, { type: string; description: string }>();

  for (const jsDoc of method.getJsDocs()) {
    for (const tag of jsDoc.getTags()) {
      if (tag.getTagName() !== "param") continue;
      const paramTag = tag as any;
      const name: string = paramTag.getName?.() ?? "";
      if (!name) continue;
      const typeExpr = paramTag.getTypeExpression?.();
      const type: string = typeExpr
        ? typeExpr.getTypeNode().getText()
        : "";
      jsDocParams.set(name, { type, description: tagComment(tag) });
    }
  }

  // Build a map of resolved property types for each parameter so we can
  // fill in types for nested @param entries (e.g. params.signer).
  const paramTypeProps = new Map<string, Map<string, string>>();
  for (const param of method.getParameters()) {
    const paramType = param.getType();
    const typeName = cleanTypeText(paramType.getText());
    const resolved = resolveTypeProperties(typeName, project);
    if (resolved) {
      paramTypeProps.set(param.getName(), resolved);
    }
  }

  const rows: ParamRow[] = [];

  for (const param of method.getParameters()) {
    const pName = param.getName();
    const typeNode = param.getTypeNode();
    const tsType = typeNode ? typeNode.getText() : "";
    const jd = jsDocParams.get(pName);

    rows.push({
      name: pName,
      type: cleanTypeText(tsType || jd?.type || ""),
      description: jd?.description ?? "",
    });

    // Nested @param entries: e.g. params.amount
    const prefix = pName + ".";
    const props = paramTypeProps.get(pName);
    for (const [key, val] of jsDocParams) {
      if (key.startsWith(prefix)) {
        const propName = key.slice(prefix.length);
        // Prefer type resolved from TS, fall back to JSDoc type
        const resolvedType = props?.get(propName) ?? val.type;
        rows.push({ name: key, type: resolvedType, description: val.description });
      }
    }
  }

  return rows;
}

function extractReturnDescription(method: MethodDeclaration): string {
  for (const jsDoc of method.getJsDocs()) {
    for (const tag of jsDoc.getTags()) {
      const tagName = tag.getTagName();
      if (tagName === "returns" || tagName === "return") {
        return tagComment(tag);
      }
    }
  }
  return "";
}

function getClassDescription(classDecl: ClassDeclaration): string {
  const docs = classDecl.getJsDocs();
  if (docs.length === 0) return "";
  return docs[docs.length - 1].getDescription().trim();
}

// ─── MDX generation ──────────────────────────────────────────────────────────

function toKebabCase(str: string): string {
  return str.replace(/([A-Z])/g, (m, c, offset) =>
    (offset > 0 ? "-" : "") + c.toLowerCase()
  );
}

function generateMdx(
  classDecl: ClassDeclaration,
  sourcePath: string,
  gitRef: string,
  githubUrlBase: string,
  sdkName: string,
  sdkPackageName: string,
  project: Project
): string {
  const className = classDecl.getName() ?? "";
  const classDesc = getClassDescription(classDecl);
  const methods = collectPublicMethods(classDecl);

  const lines: string[] = [];

  // Auto-generated warning header
  lines.push(`{/*`);
  lines.push(`  ⚠️ WARNING: DO NOT EDIT THIS FILE DIRECTLY ⚠️`);
  lines.push(``);
  lines.push(`  This file is auto-generated from the ${sdkName} source code.`);
  lines.push(``);
  lines.push(`  To update this documentation:`);
  lines.push(
    `  1. Bump the SDK version in package.json: pnpm add ${sdkPackageName}@latest`
  );
  lines.push(`  2. Run the generation script: pnpm prebuild`);
  lines.push(``);
  lines.push(`  Any manual edits will be overwritten on the next generation.`);
  lines.push(`*/}`);
  lines.push(``);

  lines.push(`## ${className}`);
  lines.push(``);
  if (classDesc) {
    lines.push(classDesc);
    lines.push(``);
  }

  // Namespace and property tables
  const allProps = collectPublicProperties(classDecl);
  const namespaces = allProps.filter((p) => p.type.includes("Namespace"));
  const properties = allProps.filter((p) => !p.type.includes("Namespace"));

  if (namespaces.length > 0) {
    lines.push(`### Namespaces`);
    lines.push(``);
    lines.push(`| Namespace | Type | Description |`);
    lines.push(`|-----------|------|-------------|`);
    for (const ns of namespaces) {
      lines.push(`| \`${ns.name}\` | \`${ns.type}\` | ${ns.description} |`);
    }
    lines.push(``);
  }

  if (properties.length > 0) {
    lines.push(`### Properties`);
    lines.push(``);
    lines.push(`| Property | Type | Description |`);
    lines.push(`|----------|------|-------------|`);
    for (const p of properties) {
      lines.push(`| \`${p.name}\` | \`${p.type}\` | ${p.description} |`);
    }
    lines.push(``);
  }

  if (methods.length > 0) {
    // Summary table
    lines.push(`### Methods`);
    lines.push(``);
    lines.push(`| Function | Description |`);
    lines.push(`|----------|-------------|`);
    for (const m of methods) {
      const anchor = m.getName().toLowerCase();
      const desc = oneLiner(m.getJsDocs()[0]?.getDescription().trim() ?? "");
      lines.push(`| **[${m.getName()}()](#${anchor})** | ${desc} |`);
    }
    lines.push(``);

    // Per-method sections
    for (const m of methods) {
      const desc = m.getJsDocs()[0]?.getDescription().trim() ?? "";
      const params = extractParams(m, project);
      const returnDesc = extractReturnDescription(m);
      const lineNumber = m.getStartLineNumber();

      // Source file may differ from the declared sourcePath when method is inherited
      const methodSourceFile = m.getSourceFile().getFilePath();
      // Try to find the relative path inside the SDK src
      const srcMarker = `${path.sep}src${path.sep}`;
      const srcIdx = methodSourceFile.lastIndexOf(srcMarker);
      const relPath =
        srcIdx >= 0
          ? `src${methodSourceFile.slice(srcIdx + srcMarker.length - 1)}`
          : sourcePath;
      // Strip .js extension that may appear in the resolved path, use .ts
      const relPathTs = relPath.replace(/\.js$/, ".ts");

      lines.push(`#### \`${m.getName()}()\``);
      lines.push(``);
      if (desc) {
        lines.push(desc);
        lines.push(``);
      }

      if (params.length > 0) {
        lines.push(`| Parameter | Type | Description |`);
        lines.push(`|-----------|------|-------------|`);
        for (const p of params) {
          const typeStr = p.type ? `\`${p.type}\`` : "";
          lines.push(`| \`${p.name}\` | ${typeStr} | ${p.description} |`);
        }
        lines.push(``);
      }

      if (returnDesc) {
        lines.push(`**Returns:** ${returnDesc}`);
        lines.push(``);
      }

      const githubUrl = `${githubUrlBase}/${relPathTs}#L${lineNumber}`;
      lines.push(
        `<sub>[<Icon icon="github" /> Source ↗](${githubUrl})</sub>`
      );
      lines.push(``);
      lines.push(`---`);
      lines.push(``);
    }
  }

  return lines.join("\n");
}

// ─── Public processComponent ─────────────────────────────────────────────────

export interface ProcessComponentParams {
  component: ComponentConfig;
  project: Project;
  sdkPath: string;
  outputDir: string;
  gitRef: string;
  githubUrlBase: string;
  sdkName: string;
  sdkPackageName: string;
}

export function processComponent(params: ProcessComponentParams): void {
  const {
    component,
    project,
    sdkPath,
    outputDir,
    gitRef,
    githubUrlBase,
    sdkName,
    sdkPackageName,
  } = params;
  const { className, sourcePath } = component;

  const fullSourcePath = path.join(sdkPath, sourcePath);
  if (!fs.existsSync(fullSourcePath)) {
    console.error(`  ✗ ${className}: source not found: ${fullSourcePath}`);
    return;
  }

  let sourceFile = project.getSourceFile(fullSourcePath);
  if (!sourceFile) {
    sourceFile = project.addSourceFileAtPath(fullSourcePath);
  }

  const classDecl = sourceFile.getClass(className);
  if (!classDecl) {
    console.error(`  ✗ ${className}: class not found in ${sourcePath}`);
    return;
  }

  const mdx = generateMdx(
    classDecl,
    sourcePath,
    gitRef,
    githubUrlBase,
    sdkName,
    sdkPackageName,
    project
  );

  const outFile = path.join(outputDir, toKebabCase(className) + ".mdx");
  fs.writeFileSync(outFile, mdx);
  console.log(`  ✓ ${className} -> ${path.basename(outFile)}`);
}
