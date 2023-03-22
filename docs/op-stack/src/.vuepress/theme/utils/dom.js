/**
 * Change DOM classes
 *
 * @param domClass DOM classlist
 * @param insert class to insert
 * @param remove class to remove
 */
export const changeClass = (domClass, insert, remove) => {
    const oldClasses = [];
    domClass.remove(...remove);
    domClass.forEach((classname) => {
        oldClasses.push(classname);
    });
    domClass.value = "";
    domClass.add(...insert, ...oldClasses);
};
//# sourceMappingURL=dom.js.map