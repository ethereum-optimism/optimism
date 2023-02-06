export const getPathMatchedKeys = (encryptOptions, path) => encryptOptions && typeof encryptOptions.config === "object"
    ? Object.keys(encryptOptions.config)
        .filter((key) => path.startsWith(key))
        .sort((a, b) => b.length - a.length)
    : [];
export const getPathEncryptStatus = (encryptOptions, passwordConfig, path) => {
    const hitKeys = getPathMatchedKeys(encryptOptions, path);
    if (hitKeys.length !== 0) {
        const { config } = encryptOptions;
        return !hitKeys.some((key) => {
            const keyConfig = config[key];
            const hitPasswords = typeof keyConfig === "string" ? [keyConfig] : keyConfig;
            return hitPasswords.some((password) => passwordConfig[key] === password);
        });
    }
    return false;
};
//# sourceMappingURL=encrypt.js.map