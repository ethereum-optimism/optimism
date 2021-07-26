import * as fs from 'fs';
import * as path from 'path';
import { World } from './World';

export function getNetworkPath(basePath: string | null, network: string, name: string, extension: string | null='json'): string {
	return path.join(basePath || '', 'networks', `${network}${name}${extension ? `.${extension}` : ''}`);
}

export async function readFile<T>(world: World | null, file: string, def: T, fn: (data: string) => T): Promise<T> {
  if (world && world.fs) {
    let data = world.fs[file];
    return Promise.resolve(data ? fn(data) : def);
  } else {
    return new Promise((resolve, reject) => {
      fs.access(file, fs.constants.F_OK, (err) => {
        if (err) {
          resolve(def);
        } else {
          fs.readFile(file, 'utf8', (err, data) => {
            return err ? reject(err) : resolve(fn(data));
          });
        }
      });
    });
  }
}

export async function writeFile<T>(world: World | null, file: string, data: string): Promise<World> {
  if (world && world.fs) {
    world = world.setIn(['fs', file], data);
    return Promise.resolve(world);
  } else {
    return new Promise((resolve, reject) => {
      fs.writeFile(file, data, (err) => {
        return err ? reject(err) : resolve(world!); // XXXS `!`
      });
    });
  }
}
