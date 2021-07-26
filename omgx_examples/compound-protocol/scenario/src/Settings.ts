import { getNetworkPath, readFile, writeFile } from './File';

export class Settings {
  basePath: string | null;
  network: string | null;
  aliases: { [name: string]: string };
  from: string | undefined;
  printTxLogs: boolean = false;

  constructor(
    basePath: string | null,
    network: string | null,
    aliases: { [name: string]: string },
    from?: string
  ) {
    this.basePath = basePath;
    this.network = network;
    this.aliases = aliases;
    this.from = from;
  }

  static deserialize(basePath: string, network: string, data: string): Settings {
    const { aliases } = JSON.parse(data);

    return new Settings(basePath, network, aliases);
  }

  serialize(): string {
    return JSON.stringify({
      aliases: this.aliases
    });
  }

  static default(basePath: string | null, network: string | null): Settings {
    return new Settings(basePath, network, {});
  }

  static getFilePath(basePath: string | null, network: string): string {
    return getNetworkPath(basePath, network, '-settings');
  }

  static load(basePath: string, network: string): Promise<Settings> {
    return readFile(null, Settings.getFilePath(basePath, network), Settings.default(basePath, network), data =>
      Settings.deserialize(basePath, network, data)
    );
  }

  async save(): Promise<void> {
    if (this.network) {
      await writeFile(null, Settings.getFilePath(this.basePath, this.network), this.serialize());
    }
  }

  lookupAlias(address: string): string {
    let entry = Object.entries(this.aliases).find(([key, value]) => {
      return value === address;
    });

    if (entry) {
      return entry[0];
    } else {
      return address;
    }
  }

  lookupAliases(address: string): string[] {
    let entries = Object.entries(this.aliases).filter(([key, value]) => {
      return value === address;
    });

    return entries.map(([key, _value]) => key);
  }

  findAlias(name: string): string | null {
    const alias = Object.entries(this.aliases).find(
      ([alias, addr]) => alias.toLowerCase() === name.toLowerCase()
    );

    if (alias) {
      return alias[1];
    } else {
      return null;
    }
  }
}
