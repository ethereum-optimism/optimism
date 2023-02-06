import type { EncryptOptions } from "../types";
export declare const getPathMatchedKeys: (encryptOptions: EncryptOptions | undefined, path: string) => string[];
export declare const getPathEncryptStatus: (encryptOptions: EncryptOptions | undefined, passwordConfig: Record<string, string>, path: string) => boolean;
