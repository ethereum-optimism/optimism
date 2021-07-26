"use strict";

const {initWorld, loadVerbose, loadInvokationOpts} = require('../scenario/.tsbuilt/World.js');
const {processEvents} = require('../scenario/.tsbuilt/CoreEvent.js');
const {parse} = require('../scenario/.tsbuilt/Parser.js');
const {ConsolePrinter} = require('../scenario/.tsbuilt/Printer.js');

const fs = require('fs');
const path = require('path');

const basePath = process.env.proj_root || path.join(process.cwd());
const baseScenarioPath = path.join(basePath, 'spec', 'scenario');
const coreMacros = fs.readFileSync(path.join(baseScenarioPath, 'CoreMacros'));

const TOTAL_GAS = 5000000;

function loadScenario(file) {
  const scenarios = {};
  const fullPath = `${baseScenarioPath}/${file}`;

  const stat = fs.statSync(fullPath);

  // Check if directory, and if so, recurse
  if (stat && stat.isDirectory()) {
    loadScenarios(fullPath);
  } else {
    // Ignore files if they don't match `.scen`
    if (file.match(/\.scen$/)) {
      // Load file data
      const data = fs.readFileSync(fullPath, 'utf8');

      // Get the name of the test from its file name
      const name = file.replace(/\..*$/g, 'Scen');

      try {
        // Try and parse the file
        const scen = parse(coreMacros + data);

        // Add each scenario, prefixed by test name
        Object.entries(scen).forEach(([key, val]) => {
          scenarios[`${name}: ${key}`] = val;
        });
      } catch (e) {
        throw `Cannot parse scenario ${file}: ${e}`
      }
    }
  }

  return scenarios;
}

function run(file) {
  const scenarios = loadScenario(file);

  /**
    * Allows user to specify a scenario filter
    */
  let scenarioFilter;

  const scenarioEnv = process.env['scenarios'] || process.env['SCENARIOS'];
  const verbose = !!process.env['verbose'];
  const network = process.env['NETWORK'] || process.env['network'] || 'test';

  if (scenarioEnv) {
    console.log(`running scenarios matching: /${scenarioEnv}/i`);
    scenarioFilter = new RegExp(scenarioEnv, 'i');
  }

  describe('ScenarioTest', () => {
    /*
     * This test runs our scenarios, which come from the reference implementation.
     */

    Object.entries(scenarios).forEach(([name, events]) => {
      if (!scenarioFilter || name.match(scenarioFilter)) {
        let fn = it;

        switch (events[0]) {
          case "Pending":
            fn = it.todo;
            events = [];
            break;
          case "Gas":
            // Skip gas tests on coverage
            if (network === 'coverage') {
              fn = it.skip;
            }
            events.shift();
            break;
          case "Only":
            fn = it.only;
            events.shift();
            break;
          case "Skip":
            fn = it.skip;
            events.shift();
            break;
        }

        if (events.length === 0) {
          fn("scenario: " + name);
        } else {
          let finalWorld;
          let runner = async () => {
            let world = await initWorld(expect, new ConsolePrinter(verbose), web3, saddle, network, accounts, basePath, TOTAL_GAS);
            world = loadVerbose(world);
            world = loadInvokationOpts(world);

            // console.log(["Scenario", name, "Events", events, world]);

            finalWorld = await processEvents(world, events);

            // console.log(["Final world", finalWorld, finalWorld.actions]);

            return finalWorld;
          }

          const spec = fn("scenario: " + name, runner, 720000);
          afterEach(() => {
            if (finalWorld)
              spec.result.description += ` [${finalWorld.gasCounter.value} wei]`;
          })
        }
      } else {
        it.skip("scenario: " + name, async () => {});
      }
    });
  });
}

module.exports = {
  run: run
};
