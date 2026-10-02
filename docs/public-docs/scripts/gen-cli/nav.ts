// docs.json navigation for gen-cli: Reference tab → one dropdown per component
// → one Mintlify version per release line → groups. The newest release line
// is the default and tagged Latest; the develop line is tagged Unreleased and
// sorts last.

import type { Component } from "./components";
import type { Page } from "./emit";

const TAB = "Reference";

interface NavGroup {
  group: string;
  pages: unknown[];
}
interface NavVersion {
  version: string;
  default?: boolean;
  tag?: string;
  groups: NavGroup[];
}
interface NavDropdown {
  dropdown: string;
  icon?: string;
  versions: NavVersion[];
}
interface NavTab {
  tab: string;
  icon?: string;
  dropdowns?: NavDropdown[];
  [k: string]: unknown;
}

export function compareLines(a: string, b: string): number {
  // Newest first; develop last.
  if (a === "develop") return 1;
  if (b === "develop") return -1;
  const pa = a.slice(1).split(".").map(Number);
  const pb = b.slice(1).split(".").map(Number);
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    const d = (pb[i] ?? 0) - (pa[i] ?? 0);
    if (d !== 0) return d;
  }
  return 0;
}

/** Build the version entry's groups from the emitted pages. */
export function versionGroups(pages: Page[]): NavGroup[] {
  const commands = pages.filter((p) => p.relPath !== "global-options.mdx").map((p) => p.url);
  const groups: NavGroup[] = [{ group: "Commands", pages: commands }];
  const global = pages.find((p) => p.relPath === "global-options.mdx");
  if (global) groups.push({ group: "Global options", pages: [global.url] });
  return groups;
}

/**
 * Splice a component's version entry into docs.json's navigation. Returns
 * true when the navigation changed. Creates the Reference tab (after the
 * Protocol tab) and the component dropdown when missing.
 */
export function spliceNav(docsJson: { navigation: { tabs: NavTab[] } }, component: Component, line: string, groups: NavGroup[]): boolean {
  const before = JSON.stringify(docsJson.navigation);
  const tabs = docsJson.navigation.tabs;
  let tab = tabs.find((t) => t.tab === TAB);
  if (!tab) {
    tab = { tab: TAB, icon: "book", dropdowns: [] };
    const protocolIdx = tabs.findIndex((t) => t.tab === "Protocol");
    tabs.splice(protocolIdx === -1 ? tabs.length : protocolIdx + 1, 0, tab);
  }
  if (!tab.dropdowns) throw new Error(`the "${TAB}" tab must hold dropdowns (one per component)`);
  let dropdown = tab.dropdowns.find((d) => d.dropdown === component.title);
  if (!dropdown) {
    dropdown = { dropdown: component.title, icon: component.icon, versions: [] };
    tab.dropdowns.push(dropdown);
    tab.dropdowns.sort((a, b) => (a.dropdown < b.dropdown ? -1 : a.dropdown > b.dropdown ? 1 : 0));
  }
  const existing = dropdown.versions.find((v) => v.version === line);
  if (existing) existing.groups = groups;
  else dropdown.versions.push({ version: line, groups });

  dropdown.versions.sort((a, b) => compareLines(a.version, b.version));
  for (const v of dropdown.versions) {
    delete v.default;
    delete v.tag;
    if (v.version === "develop") v.tag = "Unreleased";
  }
  const newest = dropdown.versions.find((v) => v.version !== "develop");
  if (newest) {
    newest.default = true;
    newest.tag = "Latest";
  }
  return JSON.stringify(docsJson.navigation) !== before;
}

/** The version entry currently in docs.json for a component line, if any. */
export function currentVersionEntry(docsJson: { navigation: { tabs: NavTab[] } }, component: Component, line: string): NavVersion | null {
  const tab = docsJson.navigation.tabs.find((t) => t.tab === TAB);
  const dropdown = tab?.dropdowns?.find((d) => d.dropdown === component.title);
  return dropdown?.versions.find((v) => v.version === line) ?? null;
}
