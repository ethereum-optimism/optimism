// ruleid: sol-style-error-format
contract SemgrepTest__sol_style_error_format__bad1 {
    // ruleid: sol-style-error-format
    error BadError();
}

// ruleid: sol-style-error-format
contract SemgrepTest__sol_style_error_format__bad2 {
    error SemgrepTest__sol_style_error_format__bad2___BadError();
}

// ok: sol-style-error-format
contract SemgrepTest__sol_style_error_format__good {
    error SemgrepTest__sol_style_error_format__good_GoodError();
}
