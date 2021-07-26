const { last } = require('./Utils/JS');
const { address, etherUnsigned } = require('./Utils/Ethereum');
const { default: diff } = require('jest-diff');
const { ComptrollerErr, TokenErr, IRErr, MathErr } = require('./Errors');

function opts(comment) {
  return {
    comment: comment,
    promise: this.promise,
    isNot: this.isNot
  };
}

function fail(msg, received=undefined) {
  return {
    pass: false,
    message: () => {
      return received === undefined ? msg : `${msg}\n\nReceived: ${this.utils.printReceived(received)}`
    }
  };
}

function logReporterFormatter(reporter) {
  return (log) => logFormatter(log, reporter);
}

function logFormatter(log, reporter=undefined) {
  return "Log {\n" + Object.entries(log).map(([key, values]) => {
    let valuesFormatted = typeof(values) === 'object'
      ?
        '\n' + Object.entries(values).map(([k, v]) => {
          let vShow = v;
          vShow = reporter && k === 'error' ? reporter.ErrorInv[v] : vShow;
          vShow = reporter && k === 'info' ? reporter.FailureInfoInv[v] : vShow;
          vShow = reporter && k === 'detail' && reporter.ErrorInv[values['error']] === 'MATH_ERROR' ? MathErr.ErrorInv[v] : vShow;

          return `\t  ${k}: ${vShow}`;
        }).join("\n")
      :
        values;

    return `\t${key}: ${valuesFormatted}`;
  }).join("\n") + "\n}";
}

function match(received, expected, pass, opts, receivedFormatter=undefined) {
  receivedFormatter = receivedFormatter || this.utils.printReceived;

  const message = pass
    ? () =>
        this.utils.matcherHint(opts.comment, undefined, undefined, opts) +
        '\n\n' +
        `Expected: ${this.utils.printExpected(expected)}\n` +
        `Received: ${receivedFormatter(received)}`
    : () => {
        const diffString = diff(expected, received, {
          expand: this.expand,
        });
        return (
          this.utils.matcherHint(opts.comment, undefined, undefined, opts) +
          '\n\n' +
          (diffString && diffString.includes('- Expect')
            ? `Difference:\n\n${diffString}`
            : `Expected: ${this.utils.printExpected(expected)}\n` +
              `Received: ${receivedFormatter(received)}`)
        );
      };

  return {
    pass: pass,
    message: message
  }
}

function solo(received, expectedThing, pass, opts, receivedFormatter=undefined) {
  receivedFormatter = receivedFormatter || this.utils.printReceived;

  const message = pass
    ? () =>
        this.utils.matcherHint(opts.comment, undefined, undefined, opts) +
        '\n\n' +
        `Expected: ${expectedThing}\n` +
        `Received: ${receivedFormatter(received)}`
    : () => {
        return (
          this.utils.matcherHint(opts.comment, undefined, undefined, opts) +
          '\n\n' +
              `Expected: ${expectedThing}\n` +
              `Received: ${receivedFormatter(received)}`
        );
      };

  return {
    pass: pass,
    message: message
  }
}

function hasError(actual, expectedErrorName, reporter) {
  let actualErrorCode = typeof(actual) === 'object' ? actual[0] : actual;
  let expectedErrorCode = reporter.Error[expectedErrorName];
  let pass = actualErrorCode == expectedErrorCode;

  return match.call(this, actual, expectedErrorName, pass, opts.call(this, `hasError[${reporter}]`))
}

function arrayEqual(a, b) {
  if (a.length !== b.length) {
    return false;
  }

  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) {
      return false;
    }
  }

  return true;
}

function objectEqual(a, b, partial=false, map=undefined) {
  return doObjectEqual(a, b, partial, map || ((x) => x));
}

function doObjectEqual(a, b, partial, map) {
  if (typeof(a) !== 'object' || typeof(b) !== 'object') {
    return a === b;
  }

  if (!partial && !arrayEqual(Object.keys(a).sort(), Object.keys(b).sort())) {
    return false;
  }

  for (let key of Object.keys(b)) {
    if (!doObjectEqual(map(a[key]), map(b[key]), false, map)) {
      return false;
    }
  }

  return true;
}

function doGetLog(result, logType, comment) {
  if (!result || !result.events) {
    return {
      failure: fail.call(this, 'Expected result object with events key', result),
      log: null
    };
  }

  let el;
  if (Array.isArray(logType)) {
    [logType, el] = logType;
  }
  let logs = Array.isArray(result.events[logType]) ? result.events[logType] : [result.events[logType]];

  const log = logs[el !== undefined ? el : logs.length - 1];
  if (!log) {
    return {
      failure: solo.call(this, result.events, `Expected log with \`${logType}\` event.`, false, opts(comment), logFormatter),
      log: null
    };
  }

  return {
    failure: null,
    log: log
  };
}

let mapValues = (l, f) => {
  return Object.entries(l).reduce((acc, [key, value]) => {
    return {...acc, [key]: f(value)}
   }, {});
  }

function hasLog(result, logType, params) {
  let {failure, log} = doGetLog.call(this, result, logType, "hasLog");
  if (failure) {
    return failure;
  }

  const returnValues = log.returnValues;

  for (let i = 0; i < Object.keys(returnValues).length; i++) {
    delete returnValues[i]; // delete log entries "0", "1", etc, since they are dups
  }

  let actual = mapValues(returnValues, (x) => x.toString());
  let expected = mapValues(params, (x) => x.toString());

  // TODO: Maybe a better success message for `isNot`?
  let pass = objectEqual(actual, expected);
  return match.call(this, actual, expected, pass, opts(`hasLog [params]`), logFormatter);
}

function hasFailure(result, expectedError, expectedInfo, expectedDetail, reporter, comment) {
  let {failure, log} = doGetLog.call(this, result, 'Failure', comment);
  if (failure) {
    return failure;
  }

  const ret = log.returnValues;
  const actual = {
    error: ret.error,
    info: ret.info,
  };

  const expected = {
    error: reporter.Error[expectedError],
    info: reporter.FailureInfo[expectedInfo]
  };

  if (expectedDetail) {
    actual.detail = ret.detail;
    expected.detail = expectedDetail;
  }

  // TODO: Better messages here
  return match.call(this, actual, expected, objectEqual(actual, expected), opts(comment), logFormatter);
}

function success(result, comment, reporter) {
  if (!result || !result.events) {
    return fail.call(this, 'Expected result object with events key', result);
  }

  const events = result.events;
  const log = last(events['Failure']);

  // TODO: Better messages here
  return solo.call(this, log, "a result with no `Failure` event", !log, opts(comment), logReporterFormatter(reporter));
}

function hasErrorTuple(result, tuple, reporter, cmp=undefined) {
  cmp = cmp || ((x, y) => x.toString() === y.toString());

  const hasErrorResult = hasError.call(this, result[0], tuple[0], reporter);
  if (!hasErrorResult.pass) {
    return hasErrorResult;
  }

  const likeResult = match.call(this, result[1], tuple[1], cmp(result[1], tuple[1]), opts.call(this, 'hasErrorTuple [first tuple]'));
  if (!likeResult.pass) {
    return likeResult;
  }

  if (tuple[2] !== undefined) {
    const like2Result = match.call(this, result[2], tuple[2], cmp(result[2], tuple[2]), opts.call(this, 'hasErrorTuple [second tuple]'));
    if (!like2Result.pass) {
      return like2Result;
    }
  }

  return match.call(this, result, tuple, true, opts('hasErrorTuple'));
}

// TODO: Improve
function revert(actual, msg) {
  return {
    pass: !!actual['message'] && actual.message === `VM Exception while processing transaction: ${msg}`,
    message: () => {
      if (actual["message"]) {
        return `expected VM Exception while processing transaction: ${msg}, got ${actual["message"]}`
      } else {
        return `expected revert, but transaction succeeded: ${JSON.stringify(actual)}`
      }
    }
  }
}

function toBeWithinDelta(received, expected, delta) {
  const pass = Math.abs(Number(received) - Number(expected)) < delta;
  if (pass) {
    return {
      message: () =>
        `expected ${received} not to be within range ${expected} ± ${delta}`,
      pass: true,
    };
  } else {
    return {
      message: () =>
        `expected ${received} to be within range ${expected} ± ${delta}`,
      pass: false,
    };
  }
};


expect.extend({
  toBeAddressZero(actual) {
    return match.call(
      this,
      actual,
      address(0),
      actual === address(0),
      opts('Expected to be zero address')
    )
  },

  toBeWithinDelta(actual, expected, delta) {
    return toBeWithinDelta.call(this, actual, expected, delta);
  },

  toHaveTrollError(actual, expectedErrorName) {
    return hasError.call(this, actual, expectedErrorName, ComptrollerErr);
  },

  toHaveTokenError(actual, expectedErrorName) {
    return hasError.call(this, actual, expectedErrorName, TokenErr);
  },

  toHaveLog(result, event, params) {
    return hasLog.call(this, result, event, params);
  },

  toHaveTrollFailure(result, err, info, detail=undefined) {
    return hasFailure.call(this, result, err, info, detail, ComptrollerErr, 'toHaveTrollFailure');
  },

  toHaveTokenFailure(result, err, info, detail=undefined) {
    return hasFailure.call(this, result, err, info, detail, TokenErr, 'toHaveTokenFailure');
  },

  toHaveTokenMathFailure(result, info, detail) {
    return hasFailure.call(this, result, 'MATH_ERROR', info, detail && (MathErr.Error[detail] || -1), TokenErr, 'toHaveTokenMathFailure');
  },

  toHaveTrollReject(result, info, detail) {
    return hasFailure.call(this, result, 'COMPTROLLER_REJECTION', info, detail && ComptrollerErr.Error[detail], TokenErr, 'toHaveTrollReject');
  },

  toHaveTrollErrorTuple(result, tuple, cmp=undefined) {
    return hasErrorTuple.call(this, result, tuple, ComptrollerErr, cmp);
  },

  toEqualNumber(actual, expected) {
    return match.call(this, actual, expected, etherUnsigned(actual).isEqualTo(etherUnsigned(expected)), opts('toEqualNumber'));
  },

  toPartEqual(actual, expected) {
    return match.call(this, actual, expected, objectEqual(actual, expected, true), opts('toPartEqual'));
  },

  toSucceed(actual) {
    return success.call(this, actual, 'success');
  },

  toTokenSucceed(actual) {
    return success.call(this, actual, 'success', TokenErr);
  },

  toRevert(actual, msg='revert') {
    return revert.call(this, actual, msg);
  },

  toRevertWithError(trx, expectedErrorName, reason='revert', reporter=TokenErr) {
    return revert.call(this, trx, `${reason} (${reporter.Error[expectedErrorName].padStart(2, '0')})`);
  },

  fail(received, msg) {
    return fail.call(this, msg, received);
  }
});
