const fs = require('fs');

(async () => {
  fs.readFile('gasCosts.json', (err, data) => {
    if (err) throw err;
    let gasReport = JSON.parse(data);

    console.log("Gas report")
    console.log("----------\n")

    for (const [scenario, report] of Object.entries(gasReport)) {
      cost = report.fee;
      console.log(scenario, "-", cost, "gas");
    }
  });
})();
