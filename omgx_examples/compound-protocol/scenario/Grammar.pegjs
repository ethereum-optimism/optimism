// See: https://pegjs.org/online

// Scenario Grammar

{
  if (!Array.prototype.flat) {
    Object.defineProperty(Array.prototype, 'flat', {
      configurable: true,
      value: function flat (x) {
        var depth = isNaN(arguments[0]) ? 1 : Number(arguments[0]);

        return depth ? Array.prototype.reduce.call(this, function (acc, cur) {
          if (Array.isArray(cur)) {
            acc.push.apply(acc, flat.call(cur, depth - 1));
          } else {
            acc.push(cur);
          }

          return acc;
        }, []) : Array.prototype.slice.call(this);
      },
      writable: true
    });
  }

  function getString(str) {
    let val;
    if (Array.isArray(str)) {
      if (str.length !== 2 || str[0] !== 'String') {
        throw new Error(`Expected string, got ${str}`);
      }

      val = str[1];
    } else {
      val = str;
    }

    if (typeof val !== 'string') {
      throw new Error(`Expected string, got ${val} (${typeof val})`);
    }

    return val;
  }

  function expandEvent(macros, step) {
    const [eventName, ...eventArgs] = step;

    if (macros[eventName]) {
      let expanded = expandMacro(macros[eventName], eventArgs);

      // Recursively expand steps
      return expanded.map(event => expandEvent(macros, event)).flat();
    } else {
      return [step];
    }
  }

  function getArgValues(eventArgs, macroArgs) {
    const eventArgNameMap = {};
    const eventArgIndexed = new Array();
    const argValues = {};
    let usedNamedArg = false;
    let usedSplat = false;

    eventArgs.forEach((eventArg) => {
      if (eventArg.argName) {
        const {argName, argValue} = eventArg;

        eventArgNameMap[argName] = argValue;
        usedNamedArg = true;
      } else {
        if (usedNamedArg) {
          throw new Error(`Cannot use positional arg after named arg in macro invokation ${JSON.stringify(eventArgs)} looking at ${eventArg.toString()}`);
        }

        eventArgIndexed.push(eventArg);
      }
    });

    macroArgs.forEach(({arg, def, splat}, argIndex) => {
      if (usedSplat) {
        throw new Error("Cannot have arg after splat arg");
      }

      let val;
      if (eventArgNameMap[arg] !== undefined) {
        val = eventArgNameMap[arg];
      } else if (splat) {
        val = eventArgIndexed.slice(argIndex); // Clear out any remaining args
        usedSplat = true;
      } else if (eventArgIndexed[argIndex] !== undefined) {
        val = eventArgIndexed[argIndex];
      } else if (def !== undefined) {
        val = def;
      } else {
        throw new Error("Macro cannot find arg value for " + arg);
      }
      argValues[arg] = val;
    });

    return argValues;
  }

  function expandMacro(macro, eventArgs) {
    const argValues = getArgValues(eventArgs, macro.args);

    function expandStep(step) {
      return step.map((token) => {
        if (argValues[token] !== undefined) {
          return argValues[token];
        } else {
          if (Array.isArray(token)) {
            return expandStep(token);
          } else {
            return token;
          }
        }
      });
    };

    return macro.steps.map(expandStep);
  }

  function addTopLevelEl(state, el) {
    const macros = state.macros;
    const tests = state.tests;
    const pending = state.pending;

    switch (el.type) {
      case 'macro':
        const macro = {[el.name]: {args: el.args, steps: el.steps}};

        return {
           tests: tests,
           macros: ({...macros, ...macro})
        };
      case 'test':
        const steps = el.steps;
        const expandedSteps = steps.map((step) => {
          return expandEvent(macros, step)
        }).flat();

        const test = {[el.test]: expandedSteps};

        return {
           tests: {...tests, ...test},
           macros: macros
        }
    }
  }
}

tests
  = values:(
      head:top_level_el
      tail:(line_separator t:top_level_el { return t; })*
      { return tail.reduce((acc, el) => addTopLevelEl(acc, el), addTopLevelEl({macros: {}, tests: {}}, head)); }
    )?
    full_ws
    { return values !== null ? values.tests : {}; }

macros
  = values:(
      head:top_level_el
      tail:(line_separator t:top_level_el { return t; })*
      { return tail.reduce((acc, el) => addTopLevelEl(acc, el), addTopLevelEl({macros: {}, tests: {}}, head)); }
    )?
    full_ws
    { return values !== null ? values.macros : {}; }

top_level_el
  = test
  / macro
  / gastest
  / pending
  / only
  / skip

test
  = full_ws? "Test" ws name:string ws line_separator steps:steps? { return {type: 'test', test: getString(name), steps: steps}; }

gastest
  = full_ws? "GasTest" ws name:string ws line_separator steps:steps? { return {type: 'test', test: getString(name), steps: ["Gas"].concat(steps)}; }

pending
  = full_ws? "Pending" ws name:string ws line_separator steps:steps? { return {type: 'test', test: getString(name), steps: ["Pending"].concat(steps)}; }

only
  = full_ws? "Only" ws name:string ws line_separator steps:steps? { return {type: 'test', test: getString(name), steps: ["Only"].concat(steps)}; }

skip
  = full_ws? "Skip" ws name:string ws line_separator steps:steps? { return {type: 'test', test: getString(name), steps: ["Skip"].concat(steps)}; }

macro
  = full_ws? "Macro" ws name:token ws args:args? line_separator steps:steps { return {type: 'macro', name: getString(name), args: args || [], steps: steps}; }

args
  = args:(
      head:arg
      tail:(ws t:args { return t; })*
      { return [head].concat(tail).filter((x) => !!x); }
  )
  { return args !== null ? args.flat() : []; }

arg
  = splat:("..."?) arg:token def:(ws? "=" t:token ws? { return t; })? { return { arg, def, splat }; }

token_set
  = tokens:(
      head:token
      tail:(ws t:token_set { return t; })*
      { return [head].concat(tail).filter((x) => !!x); }
  )
  { return tokens !== null ? tokens.flat() : []; }

steps
  = steps:(
      head:full_expr
      tail:(line_separator step:full_expr { return step; })*
      { return [head].concat(tail).filter((x) => !!x); }
    )?
    { return steps !== null ? steps : []; }

full_expr
  = tab_separator step:step { return step; }
  / comment { return null; }
  / tab_separator ws { return null; }

step
  = val:expr comment? { return val; }
  / comment { return null; }
  / tab_separator? ws { return null; }

expr
  = (
      head:token
        tail:(ws continuation? value:expr { return value })*
        { return [head].concat(tail.flat(1)); }
    )
    / begin_compound inner:expr end_compound { return [inner]; }
    / begin_list inner:list_inner? end_list { return [["List"].concat((inner || []).flat())] };

comment
  = ws "--" [^\n]* { return null; }
  / ws "#" [^\n]* { return null; }

token =
  token1:simple_token ":" token2:simple_token { return {argName: token1, argValue: token2} }
  / simple_token

simple_token =
    hex
  / number
  / ( t:([A-Za-z0-9_]+) { return t.join("") } )
  / string

hex = hex:("0x" [0-9a-fA-F]+)  { return ["Hex", hex.flat().flat().join("")] }
number =
  n:(("-" / "+")? [0-9]+ ("." [0-9]+)? ("e" "-"? [0-9]+)?) { return ["Exactly", n.flat().flat().join("")] }

list_inner
  = (
      head:expr
        tail:(ws? value:list_inner { return value })*
        { return [head].concat(tail.flat()); }
    )

begin_compound    = ws "(" ws
end_compound    = ws ")" ws

begin_list    = ws "[" ws
end_list    = ws "]" ws

line_separator  = "\r"?"\n"
tab_separator = "\t"
              / "    "

continuation = "\\" line_separator tab_separator tab_separator

ws "whitespace" = [ \t]*
full_ws = comment full_ws
    / [ \t\r\n] full_ws?

string "string"
  = quotation_mark chars:char* quotation_mark { return ["String", chars.join("")]; }

char
  = unescaped
  / escape
    sequence:(
        '"'
      / "\\"
      / "/"
      / "b" { return "\b"; }
      / "f" { return "\f"; }
      / "n" { return "\n"; }
      / "r" { return "\r"; }
      / "t" { return "\t"; }
      / "u" digits:$(HEXDIG HEXDIG HEXDIG HEXDIG) {
          return String.fromCharCode(parseInt(digits, 16));
        }
    )
    { return sequence; }

escape
  = "\\"

quotation_mark
  = '"'

unescaped
  = [^\0-\x1F\x22\x5C]

// ----- Core ABNF Rules -----

// See RFC 4234, Appendix B (http://tools.ietf.org/html/rfc4234).
DIGIT  = [0-9]
HEXDIG = [0-9a-f]i