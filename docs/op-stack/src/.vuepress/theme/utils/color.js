export default class Color {
    constructor(type, red, green, blue, alpha = 1) {
        this.type = type;
        this.red = red;
        this.green = green;
        this.blue = blue;
        this.alpha = alpha;
    }
    static fromHex(color) {
        const parseHex = (colorString) => parseInt(colorString, 16);
        const parseAlpha = (colorString, total) => Math.round((parseHex(colorString) * 100) / total) / 100;
        if (color.length === 4)
            return new Color("hex", parseHex(color[1]) * 17, parseHex(color[2]) * 17, parseHex(color[3]) * 17);
        if (color.length === 5)
            return new Color("hex", parseHex(color[1]) * 17, parseHex(color[2]) * 17, parseHex(color[3]) * 17, parseAlpha(color[4], 15));
        if (color.length === 7)
            return new Color("hex", parseHex(color.substring(1, 3)), parseHex(color.substring(3, 5)), parseHex(color.substring(5, 7)));
        return new Color("hex", parseHex(color.substring(1, 3)), parseHex(color.substring(3, 5)), parseHex(color.substring(5, 7)), parseAlpha(color.substring(7, 9), 255));
    }
    // From RGB or RGBA
    static fromRGB(color) {
        // eslint-disable-next-line @typescript-eslint/naming-convention
        const RGBAPattern = /rgba\((.+)?,(.+)?,(.+)?,(.+)?\)/u;
        // eslint-disable-next-line @typescript-eslint/naming-convention
        const RGBPattern = /rgb\((.+)?,(.+)?,(.+)?\)/u;
        const fromRGB = (colorString) => colorString.includes("%")
            ? (Number(colorString.trim().substring(0, colorString.trim().length - 1)) /
                100) *
                256 -
                1
            : Number(colorString.trim());
        const rgbaResult = RGBAPattern.exec(color);
        if (rgbaResult)
            return new Color("rgb", fromRGB(rgbaResult[1]), fromRGB(rgbaResult[2]), fromRGB(rgbaResult[3]), Number(rgbaResult[4] || 1));
        const rgbResult = RGBPattern.exec(color);
        if (rgbResult)
            return new Color("rgb", fromRGB(rgbResult[1]), fromRGB(rgbResult[2]), fromRGB(rgbResult[3]));
        throw new Error(`Can not handle color: ${color}`);
    }
    static getColor(colorString) {
        if (colorString.startsWith("#"))
            return this.fromHex(colorString);
        return this.fromRGB(colorString);
    }
    toString() {
        if (this.type === "hex" && this.alpha === 1) {
            const toHex = (color) => color < 10
                ? color.toString()
                : color === 10
                    ? "a"
                    : color === 11
                        ? "b"
                        : color === 12
                            ? "c"
                            : color === 13
                                ? "d"
                                : color === 14
                                    ? "e"
                                    : "f";
            if (this.red % 17 === 0 && this.green % 17 === 0 && this.blue % 17 === 0)
                return `#${toHex(this.red / 17)}${toHex(this.green / 17)}${toHex(this.blue / 17)}`;
            const getHex = (color) => toHex((color - (color % 16)) / 16) + toHex(color % 16);
            return `#${getHex(this.red)}${getHex(this.green)}${getHex(this.blue)}`;
        }
        return this.alpha === 1
            ? `rgb(${this.red},${this.green},${this.blue})`
            : `rgba(${this.red},${this.green},${this.blue},${this.alpha})`;
    }
    adjust(item, amount) {
        const result = Math.round(this[item] * amount);
        if (item === "alpha")
            this.alpha = result < 0 ? 0 : result > 1 ? 1 : result;
        else
            this[item] = result < 0 ? 0 : result > 255 ? 255 : result;
    }
    darken(amount) {
        this.adjust("red", 1 - amount);
        this.adjust("green", 1 - amount);
        this.adjust("blue", 1 - amount);
        return this;
    }
    lighten(amount) {
        this.adjust("red", 1 + amount);
        this.adjust("green", 1 + amount);
        this.adjust("blue", 1 + amount);
        return this;
    }
}
//# sourceMappingURL=color.js.map