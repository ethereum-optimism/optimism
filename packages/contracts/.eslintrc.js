module.exports = {
    "extends": "../../.eslintrc.js",
    "parserOptions": {
        "project": "tsconfig.json",
        "sourceType": "module"
    },
    "rules": {
        "@typescript-eslint/no-empty-function": "off",
        "no-empty-function": "off",
        "prefer-arrow/prefer-arrow-functions": "off",
        "jsdoc/newline-after-description": "off",
        "no-shadow": "off",
        "jsdoc/check-indentation": "off",
        "@typescript-eslint/no-shadow": "off",
        "@typescript-eslint/no-var-requires": "off",
    },
}