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
        pageSize: 20,
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
                            <th scope="col">L2Head</th>
                            <th scope="col">L2SafeHead</th>
                            <th scope="col">L2FinalizedHead</th>
                            <th scope="col">L1WindowBuf</th>
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

                    let windowBufEl = `<ul style="list-style-type:none">`
                    e.l1WindowBuf.forEach((x) => {
                        windowBufEl += `<li title=${JSON.stringify(x)} data-toggle="tooltip" style="background-color:${colorCode(x.hash)};">${prettyHex(x.hash)}</li>`
                    })
                    windowBufEl += "</ul>"

                    // TODO: click to copy full hash
                    html += `<tr>
                        <td title="${e.event}" data-toggle="tooltip">
                            ${e.t}
                        </td>
                        <td title=${JSON.stringify(e.l1Head)} data-toggle="tooltip" style="background-color:${colorCode(e.l1Head.hash)};">
                            ${prettyHex(e.l1Head.hash)}
                        </td>
                        <td title=${JSON.stringify(e.l2Head)} data-toggle="tooltip" style="background-color:${colorCode(e.l2Head.hash)};">
                            ${prettyHex(e.l2Head.hash)}
                        </td>
                        <td title=${JSON.stringify(e.l2SafeHead)} data-toggle="tooltip" style="background-color:${colorCode(e.l2SafeHead.hash)};">
                            ${prettyHex(e.l2SafeHead.hash)}
                        </td>
                        <td title=${JSON.stringify(e.l2FinalizedHead)} data-toggle="tooltip" style="background-color:${colorCode(e.l2FinalizedHead.hash)};">
                            ${prettyHex(e.l2FinalizedHead.hash)}
                        </td>
                        <td>${windowBufEl}</td>
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
