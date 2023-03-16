/** Group lower level headings under h2 children */
export const groupHeaders = (headers) => {
    const headerscopy = headers.map((header) => (Object.assign({}, header)));
    let lastH2;
    // group other headings under h2 headings
    headerscopy.forEach((header) => {
        if (header.level === 2)
            lastH2 = header;
        else if (lastH2) {
            if (!lastH2.children)
                lastH2.children = [];
            lastH2.children.push(header);
        }
    });
    // filter only h2 headings
    return headerscopy.filter((header) => header.level === 2);
};
//# sourceMappingURL=groupHeader.js.map