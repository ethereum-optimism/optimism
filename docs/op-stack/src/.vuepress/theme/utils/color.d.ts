export default class Color {
    type: "hex" | "rgb";
    red: number;
    green: number;
    blue: number;
    alpha: number;
    constructor(type: "hex" | "rgb", red: number, green: number, blue: number, alpha?: number);
    static fromHex(color: string): Color;
    static fromRGB(color: string): Color;
    static getColor(colorString: string): Color;
    toString(): string;
    adjust(item: "red" | "green" | "blue" | "alpha", amount: number): void;
    darken(amount: number): Color;
    lighten(amount: number): Color;
}
