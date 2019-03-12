/// <reference types="node" />
export interface ABIResult {
    [key: number]: string | number | boolean;
}
interface ABIType {
    type: string;
    name: string;
    components?: ABIType[];
}
export interface ABIObject {
    type: string;
    name: string;
    inputs: ABIType[];
    outputs: ABIType[];
}
export declare const encodeParams: (types: string[], args: any[]) => string;
export declare const encodeMethod: (methodAbi: ABIObject, args: any[]) => string;
export declare const decodeResponse: (methodAbi: ABIObject, response: Uint8Array | Buffer) => ABIResult;
export {};
