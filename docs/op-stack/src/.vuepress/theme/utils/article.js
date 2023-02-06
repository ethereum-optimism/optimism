import * as dayjs from "dayjs";
export const getDate = (date) => {
    const time = dayjs(date instanceof Date || typeof date === "number" ? date : date.trim());
    if (time.isValid()) {
        const year = time.year();
        const month = time.month() + 1;
        const date = time.date();
        const hour = time.hour();
        const minute = time.minute();
        const second = time.second();
        const millisecond = time.millisecond();
        if ((hour === 8 || hour === 0) &&
            minute === 0 &&
            second === 0 &&
            millisecond === 0)
            return [year, month, date, undefined, undefined, undefined];
        return [year, month, date, hour, minute, second];
    }
    const pattern = /(?:(\d+)[/-](\d+)[/-](\d+))?\s*(?:(\d+):(\d+)(?::(\d+))?)?/u;
    const [, year, month, day, hour, minute, second] = pattern.exec(date.trim()) || [];
    const getNumber = (a) => typeof a === "undefined" ? undefined : Number(a);
    const getYear = (yearNumber) => yearNumber && yearNumber < 100 ? yearNumber + 2000 : yearNumber;
    const getSecond = (secondNumber) => hour && minute && !second ? 0 : secondNumber;
    return [
        getYear(getNumber(year)),
        getNumber(month),
        getNumber(day),
        getNumber(hour),
        getNumber(minute),
        getSecond(getNumber(second)),
    ];
};
export const compareDate = (dataA, dataB) => {
    if (!dataA)
        return 1;
    if (!dataB)
        return -1;
    const compare = (a, b) => {
        if (a.length === 0)
            return 0;
        if (typeof b[0] === "undefined")
            return typeof a[0] === "undefined" || a[0] === 0 ? 0 : -1;
        if (typeof a[0] === "undefined")
            return b[0] === 0 ? 0 : 1;
        if (b[0] - a[0] === 0) {
            a.shift();
            b.shift();
            return compare(a, b);
        }
        return b[0] - a[0];
    };
    return compare(getDate(dataA), getDate(dataB));
};
export const filterArticle = (pages, filterFunc) => pages.filter((page) => {
    const { frontmatter: { article, blogpage, home }, title, } = page;
    return (typeof title !== "undefined" &&
        blogpage !== true &&
        home !== true &&
        article !== false &&
        (!filterFunc || filterFunc(page)));
});
export const sortArticle = (pages, compareKey) => pages.slice(0).sort((prev, next) => {
    if (compareKey) {
        const prevKey = prev.frontmatter[compareKey];
        const nextKey = next.frontmatter[compareKey];
        if (prevKey && nextKey && prevKey !== nextKey)
            return Number(nextKey) - Number(prevKey);
        if (prevKey && !nextKey)
            return -1;
        if (!prevKey && nextKey)
            return 1;
    }
    const prevTime = prev.frontmatter.time || prev.frontmatter.date || prev.createTimeStamp;
    const nextTime = next.frontmatter.time || next.frontmatter.date || next.createTimeStamp;
    return compareDate(prevTime, nextTime);
});
export const generatePagination = (pages, perPage = 10) => {
    const result = [];
    let index = 0;
    while (index < pages.length) {
        const paginationPage = [];
        for (let i = 0; i < perPage; i++)
            if (index < pages.length) {
                paginationPage.push(pages[index]);
                index += 1;
            }
        result.push(paginationPage);
    }
    return result;
};
//# sourceMappingURL=article.js.map