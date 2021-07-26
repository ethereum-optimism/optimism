import * as readline from 'readline';
import * as fs from 'fs';
import {readFile} from './File';

let readlineAny = <any>readline;

export async function createInterface(options): Promise<readline.ReadLine> {
	let history: string[] = await readFile(null, options['path'], [], (x) => x.split("\n"));
	let cleanHistory = history.filter((x) => !!x).reverse();

	readlineAny.kHistorySize = Math.max(readlineAny.kHistorySize, options['maxLength']);

	let rl = readline.createInterface(options);
	let rlAny = <any>rl;

	let oldAddHistory = rlAny._addHistory;

	rlAny._addHistory = function() {
		let last = rlAny.history[0];
		let line = oldAddHistory.call(rl);

		// TODO: Should this be sync?
		if (line.length > 0 && line != last) {
			fs.appendFileSync(options['path'], `${line}\n`);
		}

		// TODO: Truncate file?

		return line;
	}

	rlAny.history.push.apply(rlAny.history, cleanHistory);

	return rl;
}
