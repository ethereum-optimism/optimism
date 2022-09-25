function prettyHex(hash) {
    return `${hash.slice(0, 8)}..${hash.slice(56)}`;
}

// return a light-ish hue
function colorCode(hash) {
    const code = parseInt(hash.slice(60), 16);
    const h = code % 361;
    const s = code % 101;
    let l = code % 101;
    // yeah this is biased but it's good enough
    if (l < 30) {
        l += 30;
    }
    return `hsl(${h}, ${s}%, ${l}%)`;
}

async function fetchLogs() {
    const response = await fetch("/logs");
    return await response.json();
}

function tooltipFormat(v) {
  var out = ""
  out += `<div>`
  out += `<em>hash</em>: <code>${v["hash"]}</code><br/>`
  out += `<em>num</em>: <code>${v["number"]}</code><br/>`
  out += `<em>parent</em>: <code>${v["parentHash"]}</code><br/>`
  out += `<em>time</em>: <code>${v["timestamp"]}</code><br/>`
  if(v.hasOwnProperty("l1origin")) {
    out += `<em>L1 hash</em>: <code>${v["l1origin"]["hash"]}</code><br/>`
    out += `<em>L1 num</em>: <code>${v["l1origin"]["number"]}</code><br/>`
    out += `<em>seq</em>: <code>${v["sequenceNumber"]}</code><br/>`
  }
  out += `</div>`
  return out
}

async function pageTable() {
    const logs = await fetchLogs();
    if (logs.length === 0) {
        return
    }

    const dataEl = $(`<div id="snapshot-tables" class="row"></div>`);
    $("#logs").append(dataEl);

    let numCols = 0
    if (logs.length !== 0) {
        numCols = logs[0].length
    }
    const paginationEl = $(`<div id="pagination"></div>`)
    $("#logs").append(paginationEl)
    paginationEl.pagination({
        dataSource: logs,
        pageSize: 40,
        showGoInput: true,
        showGoButton: true,
        callback: (data, pagination) => {
            let tables = []
            for (var i = 0; i < numCols; i++) {
                // TODO: Fix grid overflow with more than 2 rollup drivers
                let html = '<div class="col-6">';
                html += `<table class="table">
                    <thead>
                    <caption style="caption-side:top">${data[0][i].engine_addr}</caption>
                        <tr>
                            <th scope="col">Timestamp</th>
                            <th scope="col">L1Head</th>
                            <th scope="col">L1Current</th>
                            <th scope="col">L2Head</th>
                            <th scope="col">L2Safe</th>
                            <th scope="col">L2FinalizedHead</th>
                        </tr>
                    </thead>
                        `;
                html += "<tbody>";

                // TODO: it'll also be useful to indicate which rollup driver updated its state for the given timestamp
                for (const record of data) {
                    const e = record[i];
                    if (e === undefined) {
                        // this column has reached its end
                        break
                    }
                    // outer stringify in title attribute escapes the content and adds the quotes for the html to be valid
                    // inner stringify in

                    // TODO: click to copy full hash
                    html += `<tr>
                        <td title="${e.event}" data-toggle="tooltip">
                            ${e.t}
                        </td>
                        <td title="${tooltipFormat(e.l1Head)}" data-bs-html="true" data-toggle="tooltip" style="background-color:${colorCode(e.l1Head.hash)};">
                            ${prettyHex(e.l1Head.hash)}
                        </td>
                        <td title="${tooltipFormat(e.l1Current)}" data-bs-html="true" data-toggle="tooltip" style="background-color:${colorCode(e.l1Current.hash)};">
                            ${prettyHex(e.l1Current.hash)}
                        </td>
                        <td title="${tooltipFormat(e.l2Head)}" data-bs-html="true" data-toggle="tooltip" style="background-color:${colorCode(e.l2Head.hash)};">
                            ${prettyHex(e.l2Head.hash)}
                        </td>
                        <td title="${tooltipFormat(e.l2Safe)}" data-bs-html="true" data-toggle="tooltip" style="background-color:${colorCode(e.l2Safe.hash)};">
                            ${prettyHex(e.l2Safe.hash)}
                        </td>
                        <td title="${tooltipFormat(e.l2FinalizedHead)}" data-bs-html="true" data-toggle="tooltip" style="background-color:${colorCode(e.l2FinalizedHead.hash)};">
                            ${prettyHex(e.l2FinalizedHead.hash)}
                        </td>
                    </tr>`;
                }
                html += "</tbody>";
                html += "</table></div>";
                tables.push(html);
            }

            const html = tables.join("\n");
            dataEl.html(html);
            $('[data-toggle="tooltip"]').tooltip();
        }
    })
}

(async () => {
    pageTable()
})()
