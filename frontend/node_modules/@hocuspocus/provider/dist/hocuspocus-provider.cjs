'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var common = require('@hocuspocus/common');
var Y = require('yjs');
var attempt = require('@lifeomic/attempt');

function _interopNamespace(e) {
  if (e && e.__esModule) return e;
  var n = Object.create(null);
  if (e) {
    Object.keys(e).forEach(function (k) {
      if (k !== 'default') {
        var d = Object.getOwnPropertyDescriptor(e, k);
        Object.defineProperty(n, k, d.get ? d : {
          enumerable: true,
          get: function () { return e[k]; }
        });
      }
    });
  }
  n["default"] = e;
  return Object.freeze(n);
}

var Y__namespace = /*#__PURE__*/_interopNamespace(Y);

/**
 * Utility module to work with key-value stores.
 *
 * @module map
 */

/**
 * Creates a new Map instance.
 *
 * @function
 * @return {Map<any, any>}
 *
 * @function
 */
const create$2 = () => new Map();

/**
 * Get map property. Create T if property is undefined and set T on map.
 *
 * ```js
 * const listeners = map.setIfUndefined(events, 'eventName', set.create)
 * listeners.add(listener)
 * ```
 *
 * @function
 * @template V,K
 * @template {Map<K,V>} MAP
 * @param {MAP} map
 * @param {K} key
 * @param {function():V} createT
 * @return {V}
 */
const setIfUndefined = (map, key, createT) => {
  let set = map.get(key);
  if (set === undefined) {
    map.set(key, set = createT());
  }
  return set
};

/**
 * Utility module to work with sets.
 *
 * @module set
 */

const create$1 = () => new Set();

/**
 * Utility module to work with Arrays.
 *
 * @module array
 */

/**
 * Transforms something array-like to an actual Array.
 *
 * @function
 * @template T
 * @param {ArrayLike<T>|Iterable<T>} arraylike
 * @return {T}
 */
const from = Array.from;

/**
 * Utility module to work with strings.
 *
 * @module string
 */

const fromCharCode = String.fromCharCode;

/**
 * @param {string} s
 * @return {string}
 */
const toLowerCase = s => s.toLowerCase();

const trimLeftRegex = /^\s*/g;

/**
 * @param {string} s
 * @return {string}
 */
const trimLeft = s => s.replace(trimLeftRegex, '');

const fromCamelCaseRegex = /([A-Z])/g;

/**
 * @param {string} s
 * @param {string} separator
 * @return {string}
 */
const fromCamelCase = (s, separator) => trimLeft(s.replace(fromCamelCaseRegex, match => `${separator}${toLowerCase(match)}`));

/**
 * @param {string} str
 * @return {Uint8Array}
 */
const _encodeUtf8Polyfill = str => {
  const encodedString = unescape(encodeURIComponent(str));
  const len = encodedString.length;
  const buf = new Uint8Array(len);
  for (let i = 0; i < len; i++) {
    buf[i] = /** @type {number} */ (encodedString.codePointAt(i));
  }
  return buf
};

/* c8 ignore next */
const utf8TextEncoder = /** @type {TextEncoder} */ (typeof TextEncoder !== 'undefined' ? new TextEncoder() : null);

/**
 * @param {string} str
 * @return {Uint8Array}
 */
const _encodeUtf8Native = str => utf8TextEncoder.encode(str);

/**
 * @param {string} str
 * @return {Uint8Array}
 */
/* c8 ignore next */
const encodeUtf8 = utf8TextEncoder ? _encodeUtf8Native : _encodeUtf8Polyfill;

/* c8 ignore next */
let utf8TextDecoder = typeof TextDecoder === 'undefined' ? null : new TextDecoder('utf-8', { fatal: true, ignoreBOM: true });

/* c8 ignore start */
if (utf8TextDecoder && utf8TextDecoder.decode(new Uint8Array()).length === 1) {
  // Safari doesn't handle BOM correctly.
  // This fixes a bug in Safari 13.0.5 where it produces a BOM the first time it is called.
  // utf8TextDecoder.decode(new Uint8Array()).length === 1 on the first call and
  // utf8TextDecoder.decode(new Uint8Array()).length === 1 on the second call
  // Another issue is that from then on no BOM chars are recognized anymore
  /* c8 ignore next */
  utf8TextDecoder = null;
}

/**
 * Often used conditions.
 *
 * @module conditions
 */

/**
 * @template T
 * @param {T|null|undefined} v
 * @return {T|null}
 */
/* c8 ignore next */
const undefinedToNull = v => v === undefined ? null : v;

/* eslint-env browser */

/**
 * Isomorphic variable storage.
 *
 * Uses LocalStorage in the browser and falls back to in-memory storage.
 *
 * @module storage
 */

/* c8 ignore start */
class VarStoragePolyfill {
  constructor () {
    this.map = new Map();
  }

  /**
   * @param {string} key
   * @param {any} newValue
   */
  setItem (key, newValue) {
    this.map.set(key, newValue);
  }

  /**
   * @param {string} key
   */
  getItem (key) {
    return this.map.get(key)
  }
}
/* c8 ignore stop */

/**
 * @type {any}
 */
let _localStorage = new VarStoragePolyfill();
let usePolyfill = true;

/* c8 ignore start */
try {
  // if the same-origin rule is violated, accessing localStorage might thrown an error
  if (typeof localStorage !== 'undefined' && localStorage) {
    _localStorage = localStorage;
    usePolyfill = false;
  }
} catch (e) { }
/* c8 ignore stop */

/**
 * This is basically localStorage in browser, or a polyfill in nodejs
 */
/* c8 ignore next */
const varStorage = _localStorage;

/**
 * A polyfill for `addEventListener('storage', event => {..})` that does nothing if the polyfill is being used.
 *
 * @param {function({ key: string, newValue: string, oldValue: string }): void} eventHandler
 * @function
 */
/* c8 ignore next */
const onChange = eventHandler => usePolyfill || addEventListener('storage', /** @type {any} */ (eventHandler));

/**
 * A polyfill for `removeEventListener('storage', event => {..})` that does nothing if the polyfill is being used.
 *
 * @param {function({ key: string, newValue: string, oldValue: string }): void} eventHandler
 * @function
 */
/* c8 ignore next */
const offChange = eventHandler => usePolyfill || removeEventListener('storage', /** @type {any} */ (eventHandler));

/**
 * Utility functions for working with EcmaScript objects.
 *
 * @module object
 */

/**
 * @param {Object<string,any>} obj
 */
const keys = Object.keys;

/**
 * @todo implement mapToArray & map
 *
 * @template R
 * @param {Object<string,any>} obj
 * @param {function(any,string):R} f
 * @return {Array<R>}
 */
const map = (obj, f) => {
  const results = [];
  for (const key in obj) {
    results.push(f(obj[key], key));
  }
  return results
};

/**
 * @param {Object<string,any>} obj
 * @return {number}
 */
const length$1 = obj => keys(obj).length;

/**
 * Calls `Object.prototype.hasOwnProperty`.
 *
 * @param {any} obj
 * @param {string|symbol} key
 * @return {boolean}
 */
const hasProperty = (obj, key) => Object.prototype.hasOwnProperty.call(obj, key);

/**
 * Common functions and function call helpers.
 *
 * @module function
 */

/**
 * @template T
 *
 * @param {T} a
 * @param {T} b
 * @return {boolean}
 */
const equalityStrict = (a, b) => a === b;

/* c8 ignore start */

/**
 * @param {any} a
 * @param {any} b
 * @return {boolean}
 */
const equalityDeep = (a, b) => {
  if (a == null || b == null) {
    return equalityStrict(a, b)
  }
  if (a.constructor !== b.constructor) {
    return false
  }
  if (a === b) {
    return true
  }
  switch (a.constructor) {
    case ArrayBuffer:
      a = new Uint8Array(a);
      b = new Uint8Array(b);
    // eslint-disable-next-line no-fallthrough
    case Uint8Array: {
      if (a.byteLength !== b.byteLength) {
        return false
      }
      for (let i = 0; i < a.length; i++) {
        if (a[i] !== b[i]) {
          return false
        }
      }
      break
    }
    case Set: {
      if (a.size !== b.size) {
        return false
      }
      for (const value of a) {
        if (!b.has(value)) {
          return false
        }
      }
      break
    }
    case Map: {
      if (a.size !== b.size) {
        return false
      }
      for (const key of a.keys()) {
        if (!b.has(key) || !equalityDeep(a.get(key), b.get(key))) {
          return false
        }
      }
      break
    }
    case Object:
      if (length$1(a) !== length$1(b)) {
        return false
      }
      for (const key in a) {
        if (!hasProperty(a, key) || !equalityDeep(a[key], b[key])) {
          return false
        }
      }
      break
    case Array:
      if (a.length !== b.length) {
        return false
      }
      for (let i = 0; i < a.length; i++) {
        if (!equalityDeep(a[i], b[i])) {
          return false
        }
      }
      break
    default:
      return false
  }
  return true
};

/**
 * @template V
 * @template {V} OPTS
 *
 * @param {V} value
 * @param {Array<OPTS>} options
 */
// @ts-ignore
const isOneOf = (value, options) => options.includes(value);

/**
 * Isomorphic module to work access the environment (query params, env variables).
 *
 * @module map
 */

/* c8 ignore next 2 */
// @ts-ignore
const isNode = typeof process !== 'undefined' && process.release && /node|io\.js/.test(process.release.name) && Object.prototype.toString.call(typeof process !== 'undefined' ? process : 0) === '[object process]';

/* c8 ignore next */
const isBrowser = typeof window !== 'undefined' && typeof document !== 'undefined' && !isNode;
/* c8 ignore next 3 */
typeof navigator !== 'undefined'
  ? /Mac/.test(navigator.platform)
  : false;

/**
 * @type {Map<string,string>}
 */
let params;

/* c8 ignore start */
const computeParams = () => {
  if (params === undefined) {
    if (isNode) {
      params = create$2();
      const pargs = process.argv;
      let currParamName = null;
      for (let i = 0; i < pargs.length; i++) {
        const parg = pargs[i];
        if (parg[0] === '-') {
          if (currParamName !== null) {
            params.set(currParamName, '');
          }
          currParamName = parg;
        } else {
          if (currParamName !== null) {
            params.set(currParamName, parg);
            currParamName = null;
          }
        }
      }
      if (currParamName !== null) {
        params.set(currParamName, '');
      }
      // in ReactNative for example this would not be true (unless connected to the Remote Debugger)
    } else if (typeof location === 'object') {
      params = create$2(); // eslint-disable-next-line no-undef
      (location.search || '?').slice(1).split('&').forEach((kv) => {
        if (kv.length !== 0) {
          const [key, value] = kv.split('=');
          params.set(`--${fromCamelCase(key, '-')}`, value);
          params.set(`-${fromCamelCase(key, '-')}`, value);
        }
      });
    } else {
      params = create$2();
    }
  }
  return params
};
/* c8 ignore stop */

/**
 * @param {string} name
 * @return {boolean}
 */
/* c8 ignore next */
const hasParam = (name) => computeParams().has(name);

/**
 * @param {string} name
 * @return {string|null}
 */
/* c8 ignore next 4 */
const getVariable = (name) =>
  isNode
    ? undefinedToNull(process.env[name.toUpperCase()])
    : undefinedToNull(varStorage.getItem(name));

/**
 * @param {string} name
 * @return {boolean}
 */
/* c8 ignore next 2 */
const hasConf = (name) =>
  hasParam('--' + name) || getVariable(name) !== null;

/* c8 ignore next */
hasConf('production');

/* c8 ignore next 2 */
const forceColor = isNode &&
  isOneOf(process.env.FORCE_COLOR, ['true', '1', '2']);

/* c8 ignore start */
!hasParam('no-colors') &&
  (!isNode || process.stdout.isTTY || forceColor) && (
  !isNode || hasParam('color') || forceColor ||
    getVariable('COLORTERM') !== null ||
    (getVariable('TERM') || '').includes('color')
);
/* c8 ignore stop */

/**
 * Common Math expressions.
 *
 * @module math
 */

const floor = Math.floor;

/**
 * @function
 * @param {number} a
 * @param {number} b
 * @return {number} The smaller element of a and b
 */
const min = (a, b) => a < b ? a : b;

/**
 * @function
 * @param {number} a
 * @param {number} b
 * @return {number} The bigger element of a and b
 */
const max = (a, b) => a > b ? a : b;

/* eslint-env browser */
const BIT7 = 64;
const BIT8 = 128;
const BITS6 = 63;
const BITS7 = 127;

/**
 * Utility helpers for working with numbers.
 *
 * @module number
 */

const MAX_SAFE_INTEGER = Number.MAX_SAFE_INTEGER;

/**
 * Efficient schema-less binary encoding with support for variable length encoding.
 *
 * Use [lib0/encoding] with [lib0/decoding]. Every encoding function has a corresponding decoding function.
 *
 * Encodes numbers in little-endian order (least to most significant byte order)
 * and is compatible with Golang's binary encoding (https://golang.org/pkg/encoding/binary/)
 * which is also used in Protocol Buffers.
 *
 * ```js
 * // encoding step
 * const encoder = encoding.createEncoder()
 * encoding.writeVarUint(encoder, 256)
 * encoding.writeVarString(encoder, 'Hello world!')
 * const buf = encoding.toUint8Array(encoder)
 * ```
 *
 * ```js
 * // decoding step
 * const decoder = decoding.createDecoder(buf)
 * decoding.readVarUint(decoder) // => 256
 * decoding.readVarString(decoder) // => 'Hello world!'
 * decoding.hasContent(decoder) // => false - all data is read
 * ```
 *
 * @module encoding
 */

/**
 * A BinaryEncoder handles the encoding to an Uint8Array.
 */
class Encoder {
  constructor () {
    this.cpos = 0;
    this.cbuf = new Uint8Array(100);
    /**
     * @type {Array<Uint8Array>}
     */
    this.bufs = [];
  }
}

/**
 * @function
 * @return {Encoder}
 */
const createEncoder = () => new Encoder();

/**
 * The current length of the encoded data.
 *
 * @function
 * @param {Encoder} encoder
 * @return {number}
 */
const length = encoder => {
  let len = encoder.cpos;
  for (let i = 0; i < encoder.bufs.length; i++) {
    len += encoder.bufs[i].length;
  }
  return len
};

/**
 * Transform to Uint8Array.
 *
 * @function
 * @param {Encoder} encoder
 * @return {Uint8Array} The created ArrayBuffer.
 */
const toUint8Array = encoder => {
  const uint8arr = new Uint8Array(length(encoder));
  let curPos = 0;
  for (let i = 0; i < encoder.bufs.length; i++) {
    const d = encoder.bufs[i];
    uint8arr.set(d, curPos);
    curPos += d.length;
  }
  uint8arr.set(new Uint8Array(encoder.cbuf.buffer, 0, encoder.cpos), curPos);
  return uint8arr
};

/**
 * Write one byte to the encoder.
 *
 * @function
 * @param {Encoder} encoder
 * @param {number} num The byte that is to be encoded.
 */
const write = (encoder, num) => {
  const bufferLen = encoder.cbuf.length;
  if (encoder.cpos === bufferLen) {
    encoder.bufs.push(encoder.cbuf);
    encoder.cbuf = new Uint8Array(bufferLen * 2);
    encoder.cpos = 0;
  }
  encoder.cbuf[encoder.cpos++] = num;
};

/**
 * Write a variable length unsigned integer. Max encodable integer is 2^53.
 *
 * @function
 * @param {Encoder} encoder
 * @param {number} num The number that is to be encoded.
 */
const writeVarUint = (encoder, num) => {
  while (num > BITS7) {
    write(encoder, BIT8 | (BITS7 & num));
    num = floor(num / 128); // shift >>> 7
  }
  write(encoder, BITS7 & num);
};

/**
 * A cache to store strings temporarily
 */
const _strBuffer = new Uint8Array(30000);
const _maxStrBSize = _strBuffer.length / 3;

/**
 * Write a variable length string.
 *
 * @function
 * @param {Encoder} encoder
 * @param {String} str The string that is to be encoded.
 */
const _writeVarStringNative = (encoder, str) => {
  if (str.length < _maxStrBSize) {
    // We can encode the string into the existing buffer
    /* c8 ignore next */
    const written = utf8TextEncoder.encodeInto(str, _strBuffer).written || 0;
    writeVarUint(encoder, written);
    for (let i = 0; i < written; i++) {
      write(encoder, _strBuffer[i]);
    }
  } else {
    writeVarUint8Array(encoder, encodeUtf8(str));
  }
};

/**
 * Write a variable length string.
 *
 * @function
 * @param {Encoder} encoder
 * @param {String} str The string that is to be encoded.
 */
const _writeVarStringPolyfill = (encoder, str) => {
  const encodedString = unescape(encodeURIComponent(str));
  const len = encodedString.length;
  writeVarUint(encoder, len);
  for (let i = 0; i < len; i++) {
    write(encoder, /** @type {number} */ (encodedString.codePointAt(i)));
  }
};

/**
 * Write a variable length string.
 *
 * @function
 * @param {Encoder} encoder
 * @param {String} str The string that is to be encoded.
 */
/* c8 ignore next */
const writeVarString = (utf8TextEncoder && /** @type {any} */ (utf8TextEncoder).encodeInto) ? _writeVarStringNative : _writeVarStringPolyfill;

/**
 * Append fixed-length Uint8Array to the encoder.
 *
 * @function
 * @param {Encoder} encoder
 * @param {Uint8Array} uint8Array
 */
const writeUint8Array = (encoder, uint8Array) => {
  const bufferLen = encoder.cbuf.length;
  const cpos = encoder.cpos;
  const leftCopyLen = min(bufferLen - cpos, uint8Array.length);
  const rightCopyLen = uint8Array.length - leftCopyLen;
  encoder.cbuf.set(uint8Array.subarray(0, leftCopyLen), cpos);
  encoder.cpos += leftCopyLen;
  if (rightCopyLen > 0) {
    // Still something to write, write right half..
    // Append new buffer
    encoder.bufs.push(encoder.cbuf);
    // must have at least size of remaining buffer
    encoder.cbuf = new Uint8Array(max(bufferLen * 2, rightCopyLen));
    // copy array
    encoder.cbuf.set(uint8Array.subarray(leftCopyLen));
    encoder.cpos = rightCopyLen;
  }
};

/**
 * Append an Uint8Array to Encoder.
 *
 * @function
 * @param {Encoder} encoder
 * @param {Uint8Array} uint8Array
 */
const writeVarUint8Array = (encoder, uint8Array) => {
  writeVarUint(encoder, uint8Array.byteLength);
  writeUint8Array(encoder, uint8Array);
};

/**
 * Error helpers.
 *
 * @module error
 */

/**
 * @param {string} s
 * @return {Error}
 */
/* c8 ignore next */
const create = s => new Error(s);

/**
 * Efficient schema-less binary decoding with support for variable length encoding.
 *
 * Use [lib0/decoding] with [lib0/encoding]. Every encoding function has a corresponding decoding function.
 *
 * Encodes numbers in little-endian order (least to most significant byte order)
 * and is compatible with Golang's binary encoding (https://golang.org/pkg/encoding/binary/)
 * which is also used in Protocol Buffers.
 *
 * ```js
 * // encoding step
 * const encoder = encoding.createEncoder()
 * encoding.writeVarUint(encoder, 256)
 * encoding.writeVarString(encoder, 'Hello world!')
 * const buf = encoding.toUint8Array(encoder)
 * ```
 *
 * ```js
 * // decoding step
 * const decoder = decoding.createDecoder(buf)
 * decoding.readVarUint(decoder) // => 256
 * decoding.readVarString(decoder) // => 'Hello world!'
 * decoding.hasContent(decoder) // => false - all data is read
 * ```
 *
 * @module decoding
 */

const errorUnexpectedEndOfArray = create('Unexpected end of array');
const errorIntegerOutOfRange = create('Integer out of Range');

/**
 * A Decoder handles the decoding of an Uint8Array.
 */
class Decoder {
  /**
   * @param {Uint8Array} uint8Array Binary data to decode
   */
  constructor (uint8Array) {
    /**
     * Decoding target.
     *
     * @type {Uint8Array}
     */
    this.arr = uint8Array;
    /**
     * Current decoding position.
     *
     * @type {number}
     */
    this.pos = 0;
  }
}

/**
 * @function
 * @param {Uint8Array} uint8Array
 * @return {Decoder}
 */
const createDecoder = uint8Array => new Decoder(uint8Array);

/**
 * Create an Uint8Array view of the next `len` bytes and advance the position by `len`.
 *
 * Important: The Uint8Array still points to the underlying ArrayBuffer. Make sure to discard the result as soon as possible to prevent any memory leaks.
 *            Use `buffer.copyUint8Array` to copy the result into a new Uint8Array.
 *
 * @function
 * @param {Decoder} decoder The decoder instance
 * @param {number} len The length of bytes to read
 * @return {Uint8Array}
 */
const readUint8Array = (decoder, len) => {
  const view = new Uint8Array(decoder.arr.buffer, decoder.pos + decoder.arr.byteOffset, len);
  decoder.pos += len;
  return view
};

/**
 * Read variable length Uint8Array.
 *
 * Important: The Uint8Array still points to the underlying ArrayBuffer. Make sure to discard the result as soon as possible to prevent any memory leaks.
 *            Use `buffer.copyUint8Array` to copy the result into a new Uint8Array.
 *
 * @function
 * @param {Decoder} decoder
 * @return {Uint8Array}
 */
const readVarUint8Array = decoder => readUint8Array(decoder, readVarUint(decoder));

/**
 * Read one byte as unsigned integer.
 * @function
 * @param {Decoder} decoder The decoder instance
 * @return {number} Unsigned 8-bit integer
 */
const readUint8 = decoder => decoder.arr[decoder.pos++];

/**
 * Read unsigned integer (32bit) with variable length.
 * 1/8th of the storage is used as encoding overhead.
 *  * numbers < 2^7 is stored in one bytlength
 *  * numbers < 2^14 is stored in two bylength
 *
 * @function
 * @param {Decoder} decoder
 * @return {number} An unsigned integer.length
 */
const readVarUint = decoder => {
  let num = 0;
  let mult = 1;
  const len = decoder.arr.length;
  while (decoder.pos < len) {
    const r = decoder.arr[decoder.pos++];
    // num = num | ((r & binary.BITS7) << len)
    num = num + (r & BITS7) * mult; // shift $r << (7*#iterations) and add it to num
    mult *= 128; // next iteration, shift 7 "more" to the left
    if (r < BIT8) {
      return num
    }
    /* c8 ignore start */
    if (num > MAX_SAFE_INTEGER) {
      throw errorIntegerOutOfRange
    }
    /* c8 ignore stop */
  }
  throw errorUnexpectedEndOfArray
};

/**
 * Read signed integer (32bit) with variable length.
 * 1/8th of the storage is used as encoding overhead.
 *  * numbers < 2^7 is stored in one bytlength
 *  * numbers < 2^14 is stored in two bylength
 * @todo This should probably create the inverse ~num if number is negative - but this would be a breaking change.
 *
 * @function
 * @param {Decoder} decoder
 * @return {number} An unsigned integer.length
 */
const readVarInt = decoder => {
  let r = decoder.arr[decoder.pos++];
  let num = r & BITS6;
  let mult = 64;
  const sign = (r & BIT7) > 0 ? -1 : 1;
  if ((r & BIT8) === 0) {
    // don't continue reading
    return sign * num
  }
  const len = decoder.arr.length;
  while (decoder.pos < len) {
    r = decoder.arr[decoder.pos++];
    // num = num | ((r & binary.BITS7) << len)
    num = num + (r & BITS7) * mult;
    mult *= 128;
    if (r < BIT8) {
      return sign * num
    }
    /* c8 ignore start */
    if (num > MAX_SAFE_INTEGER) {
      throw errorIntegerOutOfRange
    }
    /* c8 ignore stop */
  }
  throw errorUnexpectedEndOfArray
};

/**
 * We don't test this function anymore as we use native decoding/encoding by default now.
 * Better not modify this anymore..
 *
 * Transforming utf8 to a string is pretty expensive. The code performs 10x better
 * when String.fromCodePoint is fed with all characters as arguments.
 * But most environments have a maximum number of arguments per functions.
 * For effiency reasons we apply a maximum of 10000 characters at once.
 *
 * @function
 * @param {Decoder} decoder
 * @return {String} The read String.
 */
/* c8 ignore start */
const _readVarStringPolyfill = decoder => {
  let remainingLen = readVarUint(decoder);
  if (remainingLen === 0) {
    return ''
  } else {
    let encodedString = String.fromCodePoint(readUint8(decoder)); // remember to decrease remainingLen
    if (--remainingLen < 100) { // do not create a Uint8Array for small strings
      while (remainingLen--) {
        encodedString += String.fromCodePoint(readUint8(decoder));
      }
    } else {
      while (remainingLen > 0) {
        const nextLen = remainingLen < 10000 ? remainingLen : 10000;
        // this is dangerous, we create a fresh array view from the existing buffer
        const bytes = decoder.arr.subarray(decoder.pos, decoder.pos + nextLen);
        decoder.pos += nextLen;
        // Starting with ES5.1 we can supply a generic array-like object as arguments
        encodedString += String.fromCodePoint.apply(null, /** @type {any} */ (bytes));
        remainingLen -= nextLen;
      }
    }
    return decodeURIComponent(escape(encodedString))
  }
};
/* c8 ignore stop */

/**
 * @function
 * @param {Decoder} decoder
 * @return {String} The read String
 */
const _readVarStringNative = decoder =>
  /** @type any */ (utf8TextDecoder).decode(readVarUint8Array(decoder));

/**
 * Read string of variable length
 * * varUint is used to store the length of the string
 *
 * @function
 * @param {Decoder} decoder
 * @return {String} The read String
 *
 */
/* c8 ignore next */
const readVarString = utf8TextDecoder ? _readVarStringNative : _readVarStringPolyfill;

/**
 * Look ahead and read varString without incrementing position
 *
 * @function
 * @param {Decoder} decoder
 * @return {string}
 */
const peekVarString = decoder => {
  const pos = decoder.pos;
  const s = readVarString(decoder);
  decoder.pos = pos;
  return s
};

/**
 * Utility functions to work with buffers (Uint8Array).
 *
 * @module buffer
 */

/**
 * @param {number} len
 */
const createUint8ArrayFromLen = len => new Uint8Array(len);

/**
 * Create Uint8Array with initial content from buffer
 *
 * @param {ArrayBuffer} buffer
 * @param {number} byteOffset
 * @param {number} length
 */
const createUint8ArrayViewFromArrayBuffer = (buffer, byteOffset, length) => new Uint8Array(buffer, byteOffset, length);

/**
 * Create Uint8Array with initial content from buffer
 *
 * @param {ArrayBuffer} buffer
 */
const createUint8ArrayFromArrayBuffer = buffer => new Uint8Array(buffer);

/* c8 ignore start */
/**
 * @param {Uint8Array} bytes
 * @return {string}
 */
const toBase64Browser = bytes => {
  let s = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    s += fromCharCode(bytes[i]);
  }
  // eslint-disable-next-line no-undef
  return btoa(s)
};
/* c8 ignore stop */

/**
 * @param {Uint8Array} bytes
 * @return {string}
 */
const toBase64Node = bytes => Buffer.from(bytes.buffer, bytes.byteOffset, bytes.byteLength).toString('base64');

/* c8 ignore start */
/**
 * @param {string} s
 * @return {Uint8Array}
 */
const fromBase64Browser = s => {
  // eslint-disable-next-line no-undef
  const a = atob(s);
  const bytes = createUint8ArrayFromLen(a.length);
  for (let i = 0; i < a.length; i++) {
    bytes[i] = a.charCodeAt(i);
  }
  return bytes
};
/* c8 ignore stop */

/**
 * @param {string} s
 */
const fromBase64Node = s => {
  const buf = Buffer.from(s, 'base64');
  return createUint8ArrayViewFromArrayBuffer(buf.buffer, buf.byteOffset, buf.byteLength)
};

/* c8 ignore next */
const toBase64 = isBrowser ? toBase64Browser : toBase64Node;

/* c8 ignore next */
const fromBase64 = isBrowser ? fromBase64Browser : fromBase64Node;

/* eslint-env browser */

/**
 * @typedef {Object} Channel
 * @property {Set<function(any, any):any>} Channel.subs
 * @property {any} Channel.bc
 */

/**
 * @type {Map<string, Channel>}
 */
const channels = new Map();

/* c8 ignore start */
class LocalStoragePolyfill {
  /**
   * @param {string} room
   */
  constructor (room) {
    this.room = room;
    /**
     * @type {null|function({data:ArrayBuffer}):void}
     */
    this.onmessage = null;
    /**
     * @param {any} e
     */
    this._onChange = e => e.key === room && this.onmessage !== null && this.onmessage({ data: fromBase64(e.newValue || '') });
    onChange(this._onChange);
  }

  /**
   * @param {ArrayBuffer} buf
   */
  postMessage (buf) {
    varStorage.setItem(this.room, toBase64(createUint8ArrayFromArrayBuffer(buf)));
  }

  close () {
    offChange(this._onChange);
  }
}
/* c8 ignore stop */

// Use BroadcastChannel or Polyfill
/* c8 ignore next */
const BC = typeof BroadcastChannel === 'undefined' ? LocalStoragePolyfill : BroadcastChannel;

/**
 * @param {string} room
 * @return {Channel}
 */
const getChannel = room =>
  setIfUndefined(channels, room, () => {
    const subs = create$1();
    const bc = new BC(room);
    /**
     * @param {{data:ArrayBuffer}} e
     */
    /* c8 ignore next */
    bc.onmessage = e => subs.forEach(sub => sub(e.data, 'broadcastchannel'));
    return {
      bc, subs
    }
  });

/**
 * Subscribe to global `publish` events.
 *
 * @function
 * @param {string} room
 * @param {function(any, any):any} f
 */
const subscribe = (room, f) => {
  getChannel(room).subs.add(f);
  return f
};

/**
 * Unsubscribe from `publish` global events.
 *
 * @function
 * @param {string} room
 * @param {function(any, any):any} f
 */
const unsubscribe = (room, f) => {
  const channel = getChannel(room);
  const unsubscribed = channel.subs.delete(f);
  if (unsubscribed && channel.subs.size === 0) {
    channel.bc.close();
    channels.delete(room);
  }
  return unsubscribed
};

/**
 * Publish data to all subscribers (including subscribers on this tab)
 *
 * @function
 * @param {string} room
 * @param {any} data
 * @param {any} [origin]
 */
const publish = (room, data, origin = null) => {
  const c = getChannel(room);
  c.bc.postMessage(data);
  c.subs.forEach(sub => sub(data, origin));
};

/**
 * Mutual exclude for JavaScript.
 *
 * @module mutex
 */

/**
 * @callback mutex
 * @param {function():void} cb Only executed when this mutex is not in the current stack
 * @param {function():void} [elseCb] Executed when this mutex is in the current stack
 */

/**
 * Creates a mutual exclude function with the following property:
 *
 * ```js
 * const mutex = createMutex()
 * mutex(() => {
 *   // This function is immediately executed
 *   mutex(() => {
 *     // This function is not executed, as the mutex is already active.
 *   })
 * })
 * ```
 *
 * @return {mutex} A mutual exclude function
 * @public
 */
const createMutex = () => {
  let token = true;
  return (f, g) => {
    if (token) {
      token = false;
      try {
        f();
      } finally {
        token = true;
      }
    } else if (g !== undefined) {
      g();
    }
  }
};

/**
 * Utility module to work with time.
 *
 * @module time
 */

/**
 * Return current unix time.
 *
 * @return {number}
 */
const getUnixTime = Date.now;

/**
 * Observable class prototype.
 *
 * @module observable
 */

/* c8 ignore start */
/**
 * Handles named events.
 *
 * @deprecated
 * @template N
 */
class Observable {
  constructor () {
    /**
     * Some desc.
     * @type {Map<N, any>}
     */
    this._observers = create$2();
  }

  /**
   * @param {N} name
   * @param {function} f
   */
  on (name, f) {
    setIfUndefined(this._observers, name, create$1).add(f);
  }

  /**
   * @param {N} name
   * @param {function} f
   */
  once (name, f) {
    /**
     * @param  {...any} args
     */
    const _f = (...args) => {
      this.off(name, _f);
      f(...args);
    };
    this.on(name, _f);
  }

  /**
   * @param {N} name
   * @param {function} f
   */
  off (name, f) {
    const observers = this._observers.get(name);
    if (observers !== undefined) {
      observers.delete(f);
      if (observers.size === 0) {
        this._observers.delete(name);
      }
    }
  }

  /**
   * Emit a named event. All registered event listeners that listen to the
   * specified name will receive the event.
   *
   * @todo This should catch exceptions
   *
   * @param {N} name The event name.
   * @param {Array<any>} args The arguments that are applied to the event listener.
   */
  emit (name, args) {
    // copy all listeners to an array first to make sure that no event is emitted to listeners that are subscribed while the event handler is called.
    return from((this._observers.get(name) || create$2()).values()).forEach(f => f(...args))
  }

  destroy () {
    this._observers = create$2();
  }
}
/* c8 ignore end */

/**
 * @module awareness-protocol
 */

const outdatedTimeout = 30000;

/**
 * @typedef {Object} MetaClientState
 * @property {number} MetaClientState.clock
 * @property {number} MetaClientState.lastUpdated unix timestamp
 */

/**
 * The Awareness class implements a simple shared state protocol that can be used for non-persistent data like awareness information
 * (cursor, username, status, ..). Each client can update its own local state and listen to state changes of
 * remote clients. Every client may set a state of a remote peer to `null` to mark the client as offline.
 *
 * Each client is identified by a unique client id (something we borrow from `doc.clientID`). A client can override
 * its own state by propagating a message with an increasing timestamp (`clock`). If such a message is received, it is
 * applied if the known state of that client is older than the new state (`clock < newClock`). If a client thinks that
 * a remote client is offline, it may propagate a message with
 * `{ clock: currentClientClock, state: null, client: remoteClient }`. If such a
 * message is received, and the known clock of that client equals the received clock, it will override the state with `null`.
 *
 * Before a client disconnects, it should propagate a `null` state with an updated clock.
 *
 * Awareness states must be updated every 30 seconds. Otherwise the Awareness instance will delete the client state.
 *
 * @extends {Observable<string>}
 */
class Awareness extends Observable {
  /**
   * @param {Y.Doc} doc
   */
  constructor (doc) {
    super();
    this.doc = doc;
    /**
     * @type {number}
     */
    this.clientID = doc.clientID;
    /**
     * Maps from client id to client state
     * @type {Map<number, Object<string, any>>}
     */
    this.states = new Map();
    /**
     * @type {Map<number, MetaClientState>}
     */
    this.meta = new Map();
    this._checkInterval = /** @type {any} */ (setInterval(() => {
      const now = getUnixTime();
      if (this.getLocalState() !== null && (outdatedTimeout / 2 <= now - /** @type {{lastUpdated:number}} */ (this.meta.get(this.clientID)).lastUpdated)) {
        // renew local clock
        this.setLocalState(this.getLocalState());
      }
      /**
       * @type {Array<number>}
       */
      const remove = [];
      this.meta.forEach((meta, clientid) => {
        if (clientid !== this.clientID && outdatedTimeout <= now - meta.lastUpdated && this.states.has(clientid)) {
          remove.push(clientid);
        }
      });
      if (remove.length > 0) {
        removeAwarenessStates(this, remove, 'timeout');
      }
    }, floor(outdatedTimeout / 10)));
    doc.on('destroy', () => {
      this.destroy();
    });
    this.setLocalState({});
  }

  destroy () {
    this.emit('destroy', [this]);
    this.setLocalState(null);
    super.destroy();
    clearInterval(this._checkInterval);
  }

  /**
   * @return {Object<string,any>|null}
   */
  getLocalState () {
    return this.states.get(this.clientID) || null
  }

  /**
   * @param {Object<string,any>|null} state
   */
  setLocalState (state) {
    const clientID = this.clientID;
    const currLocalMeta = this.meta.get(clientID);
    const clock = currLocalMeta === undefined ? 0 : currLocalMeta.clock + 1;
    const prevState = this.states.get(clientID);
    if (state === null) {
      this.states.delete(clientID);
    } else {
      this.states.set(clientID, state);
    }
    this.meta.set(clientID, {
      clock,
      lastUpdated: getUnixTime()
    });
    const added = [];
    const updated = [];
    const filteredUpdated = [];
    const removed = [];
    if (state === null) {
      removed.push(clientID);
    } else if (prevState == null) {
      if (state != null) {
        added.push(clientID);
      }
    } else {
      updated.push(clientID);
      if (!equalityDeep(prevState, state)) {
        filteredUpdated.push(clientID);
      }
    }
    if (added.length > 0 || filteredUpdated.length > 0 || removed.length > 0) {
      this.emit('change', [{ added, updated: filteredUpdated, removed }, 'local']);
    }
    this.emit('update', [{ added, updated, removed }, 'local']);
  }

  /**
   * @param {string} field
   * @param {any} value
   */
  setLocalStateField (field, value) {
    const state = this.getLocalState();
    if (state !== null) {
      this.setLocalState({
        ...state,
        [field]: value
      });
    }
  }

  /**
   * @return {Map<number,Object<string,any>>}
   */
  getStates () {
    return this.states
  }
}

/**
 * Mark (remote) clients as inactive and remove them from the list of active peers.
 * This change will be propagated to remote clients.
 *
 * @param {Awareness} awareness
 * @param {Array<number>} clients
 * @param {any} origin
 */
const removeAwarenessStates = (awareness, clients, origin) => {
  const removed = [];
  for (let i = 0; i < clients.length; i++) {
    const clientID = clients[i];
    if (awareness.states.has(clientID)) {
      awareness.states.delete(clientID);
      if (clientID === awareness.clientID) {
        const curMeta = /** @type {MetaClientState} */ (awareness.meta.get(clientID));
        awareness.meta.set(clientID, {
          clock: curMeta.clock + 1,
          lastUpdated: getUnixTime()
        });
      }
      removed.push(clientID);
    }
  }
  if (removed.length > 0) {
    awareness.emit('change', [{ added: [], updated: [], removed }, origin]);
    awareness.emit('update', [{ added: [], updated: [], removed }, origin]);
  }
};

/**
 * @param {Awareness} awareness
 * @param {Array<number>} clients
 * @return {Uint8Array}
 */
const encodeAwarenessUpdate = (awareness, clients, states = awareness.states) => {
  const len = clients.length;
  const encoder = createEncoder();
  writeVarUint(encoder, len);
  for (let i = 0; i < len; i++) {
    const clientID = clients[i];
    const state = states.get(clientID) || null;
    const clock = /** @type {MetaClientState} */ (awareness.meta.get(clientID)).clock;
    writeVarUint(encoder, clientID);
    writeVarUint(encoder, clock);
    writeVarString(encoder, JSON.stringify(state));
  }
  return toUint8Array(encoder)
};

/**
 * @param {Awareness} awareness
 * @param {Uint8Array} update
 * @param {any} origin This will be added to the emitted change event
 */
const applyAwarenessUpdate = (awareness, update, origin) => {
  const decoder = createDecoder(update);
  const timestamp = getUnixTime();
  const added = [];
  const updated = [];
  const filteredUpdated = [];
  const removed = [];
  const len = readVarUint(decoder);
  for (let i = 0; i < len; i++) {
    const clientID = readVarUint(decoder);
    let clock = readVarUint(decoder);
    const state = JSON.parse(readVarString(decoder));
    const clientMeta = awareness.meta.get(clientID);
    const prevState = awareness.states.get(clientID);
    const currClock = clientMeta === undefined ? 0 : clientMeta.clock;
    if (currClock < clock || (currClock === clock && state === null && awareness.states.has(clientID))) {
      if (state === null) {
        // never let a remote client remove this local state
        if (clientID === awareness.clientID && awareness.getLocalState() != null) {
          // remote client removed the local state. Do not remote state. Broadcast a message indicating
          // that this client still exists by increasing the clock
          clock++;
        } else {
          awareness.states.delete(clientID);
        }
      } else {
        awareness.states.set(clientID, state);
      }
      awareness.meta.set(clientID, {
        clock,
        lastUpdated: timestamp
      });
      if (clientMeta === undefined && state !== null) {
        added.push(clientID);
      } else if (clientMeta !== undefined && state === null) {
        removed.push(clientID);
      } else if (state !== null) {
        if (!equalityDeep(state, prevState)) {
          filteredUpdated.push(clientID);
        }
        updated.push(clientID);
      }
    }
  }
  if (added.length > 0 || filteredUpdated.length > 0 || removed.length > 0) {
    awareness.emit('change', [{
      added, updated: filteredUpdated, removed
    }, origin]);
  }
  if (added.length > 0 || updated.length > 0 || removed.length > 0) {
    awareness.emit('update', [{
      added, updated, removed
    }, origin]);
  }
};

class EventEmitter {
    constructor() {
        this.callbacks = {};
    }
    on(event, fn) {
        if (!this.callbacks[event]) {
            this.callbacks[event] = [];
        }
        this.callbacks[event].push(fn);
        return this;
    }
    emit(event, ...args) {
        const callbacks = this.callbacks[event];
        if (callbacks) {
            callbacks.forEach(callback => callback.apply(this, args));
        }
        return this;
    }
    off(event, fn) {
        const callbacks = this.callbacks[event];
        if (callbacks) {
            if (fn) {
                this.callbacks[event] = callbacks.filter(callback => callback !== fn);
            }
            else {
                delete this.callbacks[event];
            }
        }
        return this;
    }
    removeAllListeners() {
        this.callbacks = {};
    }
}

/**
 * Utility module to work with urls.
 *
 * @module url
 */

/**
 * @param {Object<string,string>} params
 * @return {string}
 */
const encodeQueryParams = params =>
  map(params, (val, key) => `${encodeURIComponent(key)}=${encodeURIComponent(val)}`).join('&');

exports.MessageType = void 0;
(function (MessageType) {
    MessageType[MessageType["Sync"] = 0] = "Sync";
    MessageType[MessageType["Awareness"] = 1] = "Awareness";
    MessageType[MessageType["Auth"] = 2] = "Auth";
    MessageType[MessageType["QueryAwareness"] = 3] = "QueryAwareness";
    MessageType[MessageType["Stateless"] = 5] = "Stateless";
    MessageType[MessageType["CLOSE"] = 7] = "CLOSE";
    MessageType[MessageType["SyncStatus"] = 8] = "SyncStatus";
})(exports.MessageType || (exports.MessageType = {}));
exports.WebSocketStatus = void 0;
(function (WebSocketStatus) {
    WebSocketStatus["Connecting"] = "connecting";
    WebSocketStatus["Connected"] = "connected";
    WebSocketStatus["Disconnected"] = "disconnected";
})(exports.WebSocketStatus || (exports.WebSocketStatus = {}));

class IncomingMessage {
    constructor(data) {
        this.data = data;
        this.encoder = createEncoder();
        this.decoder = createDecoder(new Uint8Array(this.data));
    }
    peekVarString() {
        return peekVarString(this.decoder);
    }
    readVarUint() {
        return readVarUint(this.decoder);
    }
    readVarString() {
        return readVarString(this.decoder);
    }
    readVarUint8Array() {
        return readVarUint8Array(this.decoder);
    }
    writeVarUint(type) {
        return writeVarUint(this.encoder, type);
    }
    writeVarString(string) {
        return writeVarString(this.encoder, string);
    }
    writeVarUint8Array(data) {
        return writeVarUint8Array(this.encoder, data);
    }
    length() {
        return length(this.encoder);
    }
}

class HocuspocusProviderWebsocket extends EventEmitter {
    constructor(configuration) {
        super();
        this.messageQueue = [];
        this.configuration = {
            url: '',
            // @ts-ignore
            document: undefined,
            WebSocketPolyfill: undefined,
            parameters: {},
            connect: true,
            broadcast: true,
            forceSyncInterval: false,
            // TODO: this should depend on awareness.outdatedTime
            messageReconnectTimeout: 30000,
            // 1 second
            delay: 1000,
            // instant
            initialDelay: 0,
            // double the delay each time
            factor: 2,
            // unlimited retries
            maxAttempts: 0,
            // wait at least 1 second
            minDelay: 1000,
            // at least every 30 seconds
            maxDelay: 30000,
            // randomize
            jitter: true,
            // retry forever
            timeout: 0,
            onOpen: () => null,
            onConnect: () => null,
            onMessage: () => null,
            onOutgoingMessage: () => null,
            onStatus: () => null,
            onDisconnect: () => null,
            onClose: () => null,
            onDestroy: () => null,
            onAwarenessUpdate: () => null,
            onAwarenessChange: () => null,
            quiet: false,
            providerMap: new Map(),
        };
        this.webSocket = null;
        this.webSocketHandlers = {};
        this.shouldConnect = true;
        this.status = exports.WebSocketStatus.Disconnected;
        this.lastMessageReceived = 0;
        this.identifier = 0;
        this.intervals = {
            forceSync: null,
            connectionChecker: null,
        };
        this.connectionAttempt = null;
        this.receivedOnOpenPayload = undefined;
        this.receivedOnStatusPayload = undefined;
        this.closeTries = 0;
        this.setConfiguration(configuration);
        this.configuration.WebSocketPolyfill = configuration.WebSocketPolyfill
            ? configuration.WebSocketPolyfill
            : WebSocket;
        this.on('open', this.configuration.onOpen);
        this.on('open', this.onOpen.bind(this));
        this.on('connect', this.configuration.onConnect);
        this.on('message', this.configuration.onMessage);
        this.on('outgoingMessage', this.configuration.onOutgoingMessage);
        this.on('status', this.configuration.onStatus);
        this.on('status', this.onStatus.bind(this));
        this.on('disconnect', this.configuration.onDisconnect);
        this.on('close', this.configuration.onClose);
        this.on('destroy', this.configuration.onDestroy);
        this.on('awarenessUpdate', this.configuration.onAwarenessUpdate);
        this.on('awarenessChange', this.configuration.onAwarenessChange);
        this.on('close', this.onClose.bind(this));
        this.on('message', this.onMessage.bind(this));
        this.intervals.connectionChecker = setInterval(this.checkConnection.bind(this), this.configuration.messageReconnectTimeout / 10);
        if (typeof configuration.connect !== 'undefined') {
            this.shouldConnect = configuration.connect;
        }
        if (!this.shouldConnect) {
            return;
        }
        this.connect();
    }
    async onOpen(event) {
        this.receivedOnOpenPayload = event;
    }
    async onStatus(data) {
        this.receivedOnStatusPayload = data;
    }
    attach(provider) {
        let connectPromise;
        this.configuration.providerMap.set(provider.configuration.name, provider);
        if (this.status === exports.WebSocketStatus.Disconnected && this.shouldConnect) {
            connectPromise = this.connect();
        }
        if (this.receivedOnOpenPayload) {
            provider.onOpen(this.receivedOnOpenPayload);
        }
        if (this.receivedOnStatusPayload) {
            provider.onStatus(this.receivedOnStatusPayload);
        }
        return connectPromise;
    }
    detach(provider) {
        this.configuration.providerMap.delete(provider.configuration.name);
    }
    setConfiguration(configuration = {}) {
        this.configuration = { ...this.configuration, ...configuration };
    }
    async connect() {
        if (this.status === exports.WebSocketStatus.Connected) {
            return;
        }
        // Always cancel any previously initiated connection retryer instances
        if (this.cancelWebsocketRetry) {
            this.cancelWebsocketRetry();
            this.cancelWebsocketRetry = undefined;
        }
        this.receivedOnOpenPayload = undefined;
        this.receivedOnStatusPayload = undefined;
        this.shouldConnect = true;
        const abortableRetry = () => {
            let cancelAttempt = false;
            const retryPromise = attempt.retry(this.createWebSocketConnection.bind(this), {
                delay: this.configuration.delay,
                initialDelay: this.configuration.initialDelay,
                factor: this.configuration.factor,
                maxAttempts: this.configuration.maxAttempts,
                minDelay: this.configuration.minDelay,
                maxDelay: this.configuration.maxDelay,
                jitter: this.configuration.jitter,
                timeout: this.configuration.timeout,
                beforeAttempt: context => {
                    if (!this.shouldConnect || cancelAttempt) {
                        context.abort();
                    }
                },
            }).catch((error) => {
                // If we aborted the connection attempt then don’t throw an error
                // ref: https://github.com/lifeomic/attempt/blob/master/src/index.ts#L136
                if (error && error.code !== 'ATTEMPT_ABORTED') {
                    throw error;
                }
            });
            return {
                retryPromise,
                cancelFunc: () => {
                    cancelAttempt = true;
                },
            };
        };
        const { retryPromise, cancelFunc } = abortableRetry();
        this.cancelWebsocketRetry = cancelFunc;
        return retryPromise;
    }
    attachWebSocketListeners(ws, reject) {
        const { identifier } = ws;
        const onMessageHandler = (payload) => this.emit('message', payload);
        const onCloseHandler = (payload) => this.emit('close', { event: payload });
        const onOpenHandler = (payload) => this.emit('open', payload);
        const onErrorHandler = (err) => {
            reject(err);
        };
        this.webSocketHandlers[identifier] = {
            message: onMessageHandler,
            close: onCloseHandler,
            open: onOpenHandler,
            error: onErrorHandler,
        };
        const handlers = this.webSocketHandlers[ws.identifier];
        Object.keys(handlers).forEach(name => {
            ws.addEventListener(name, handlers[name]);
        });
    }
    cleanupWebSocket() {
        if (!this.webSocket) {
            return;
        }
        const { identifier } = this.webSocket;
        const handlers = this.webSocketHandlers[identifier];
        Object.keys(handlers).forEach(name => {
            var _a;
            (_a = this.webSocket) === null || _a === void 0 ? void 0 : _a.removeEventListener(name, handlers[name]);
            delete this.webSocketHandlers[identifier];
        });
        this.webSocket.close();
        this.webSocket = null;
    }
    createWebSocketConnection() {
        return new Promise((resolve, reject) => {
            if (this.webSocket) {
                this.messageQueue = [];
                this.cleanupWebSocket();
            }
            this.lastMessageReceived = 0;
            this.identifier += 1;
            // Init the WebSocket connection
            const ws = new this.configuration.WebSocketPolyfill(this.url);
            ws.binaryType = 'arraybuffer';
            ws.identifier = this.identifier;
            this.attachWebSocketListeners(ws, reject);
            this.webSocket = ws;
            // Reset the status
            this.status = exports.WebSocketStatus.Connecting;
            this.emit('status', { status: exports.WebSocketStatus.Connecting });
            // Store resolve/reject for later use
            this.connectionAttempt = {
                resolve,
                reject,
            };
        });
    }
    onMessage(event) {
        var _a;
        this.resolveConnectionAttempt();
        this.lastMessageReceived = getUnixTime();
        const message = new IncomingMessage(event.data);
        const documentName = message.peekVarString();
        (_a = this.configuration.providerMap.get(documentName)) === null || _a === void 0 ? void 0 : _a.onMessage(event);
    }
    resolveConnectionAttempt() {
        if (this.connectionAttempt) {
            this.connectionAttempt.resolve();
            this.connectionAttempt = null;
            this.status = exports.WebSocketStatus.Connected;
            this.emit('status', { status: exports.WebSocketStatus.Connected });
            this.emit('connect');
            this.messageQueue.forEach(message => this.send(message));
            this.messageQueue = [];
        }
    }
    stopConnectionAttempt() {
        this.connectionAttempt = null;
    }
    rejectConnectionAttempt() {
        var _a;
        (_a = this.connectionAttempt) === null || _a === void 0 ? void 0 : _a.reject();
        this.connectionAttempt = null;
    }
    checkConnection() {
        var _a;
        // Don’t check the connection when it’s not even established
        if (this.status !== exports.WebSocketStatus.Connected) {
            return;
        }
        // Don’t close the connection while waiting for the first message
        if (!this.lastMessageReceived) {
            return;
        }
        // Don’t close the connection when a message was received recently
        if (this.configuration.messageReconnectTimeout
            >= getUnixTime() - this.lastMessageReceived) {
            return;
        }
        // No message received in a long time, not even your own
        // Awareness updates, which are updated every 15 seconds
        // if awareness is enabled.
        this.closeTries += 1;
        // https://bugs.webkit.org/show_bug.cgi?id=247943
        if (this.closeTries > 2) {
            this.onClose({
                event: {
                    code: 4408,
                    reason: 'forced',
                },
            });
            this.closeTries = 0;
        }
        else {
            (_a = this.webSocket) === null || _a === void 0 ? void 0 : _a.close();
            this.messageQueue = [];
        }
    }
    // Ensure that the URL always ends with /
    get serverUrl() {
        while (this.configuration.url[this.configuration.url.length - 1] === '/') {
            return this.configuration.url.slice(0, this.configuration.url.length - 1);
        }
        return this.configuration.url;
    }
    get url() {
        const encodedParams = encodeQueryParams(this.configuration.parameters);
        return `${this.serverUrl}${encodedParams.length === 0 ? '' : `?${encodedParams}`}`;
    }
    disconnect() {
        this.shouldConnect = false;
        if (this.webSocket === null) {
            return;
        }
        try {
            this.webSocket.close();
            this.messageQueue = [];
        }
        catch {
            //
        }
    }
    send(message) {
        var _a;
        if (((_a = this.webSocket) === null || _a === void 0 ? void 0 : _a.readyState) === common.WsReadyStates.Open) {
            this.webSocket.send(message);
        }
        else {
            this.messageQueue.push(message);
        }
    }
    onClose({ event }) {
        this.closeTries = 0;
        this.cleanupWebSocket();
        if (this.status === exports.WebSocketStatus.Connected) {
            this.status = exports.WebSocketStatus.Disconnected;
            this.emit('status', { status: exports.WebSocketStatus.Disconnected });
            this.emit('disconnect', { event });
        }
        if (event.code === common.Unauthorized.code) {
            if (event.reason === common.Unauthorized.reason) {
                console.warn('[HocuspocusProvider] An authentication token is required, but you didn’t send one. Try adding a `token` to your HocuspocusProvider configuration. Won’t try again.');
            }
            else {
                console.warn(`[HocuspocusProvider] Connection closed with status Unauthorized: ${event.reason}`);
            }
            this.shouldConnect = false;
        }
        if (event.code === common.Forbidden.code) {
            if (!this.configuration.quiet) {
                console.warn('[HocuspocusProvider] The provided authentication token isn’t allowed to connect to this server. Will try again.');
                return; // TODO REMOVE ME
            }
        }
        if (event.code === common.MessageTooBig.code) {
            console.warn(`[HocuspocusProvider] Connection closed with status MessageTooBig: ${event.reason}`);
            this.shouldConnect = false;
        }
        if (this.connectionAttempt) {
            // That connection attempt failed.
            this.rejectConnectionAttempt();
        }
        else if (this.shouldConnect) {
            // The connection was closed by the server. Let’s just try again.
            this.connect();
        }
        // If we’ll reconnect, we’re done for now.
        if (this.shouldConnect) {
            return;
        }
        // The status is set correctly already.
        if (this.status === exports.WebSocketStatus.Disconnected) {
            return;
        }
        // Let’s update the connection status.
        this.status = exports.WebSocketStatus.Disconnected;
        this.emit('status', { status: exports.WebSocketStatus.Disconnected });
        this.emit('disconnect', { event });
    }
    destroy() {
        this.emit('destroy');
        if (this.intervals.forceSync) {
            clearInterval(this.intervals.forceSync);
        }
        clearInterval(this.intervals.connectionChecker);
        // If there is still a connection attempt outstanding then we should stop
        // it before calling disconnect, otherwise it will be rejected in the onClose
        // handler and trigger a retry
        this.stopConnectionAttempt();
        this.disconnect();
        this.removeAllListeners();
        this.cleanupWebSocket();
    }
}

/**
 * @module sync-protocol
 */

/**
 * @typedef {Map<number, number>} StateMap
 */

/**
 * Core Yjs defines two message types:
 * • YjsSyncStep1: Includes the State Set of the sending client. When received, the client should reply with YjsSyncStep2.
 * • YjsSyncStep2: Includes all missing structs and the complete delete set. When received, the client is assured that it
 *   received all information from the remote client.
 *
 * In a peer-to-peer network, you may want to introduce a SyncDone message type. Both parties should initiate the connection
 * with SyncStep1. When a client received SyncStep2, it should reply with SyncDone. When the local client received both
 * SyncStep2 and SyncDone, it is assured that it is synced to the remote client.
 *
 * In a client-server model, you want to handle this differently: The client should initiate the connection with SyncStep1.
 * When the server receives SyncStep1, it should reply with SyncStep2 immediately followed by SyncStep1. The client replies
 * with SyncStep2 when it receives SyncStep1. Optionally the server may send a SyncDone after it received SyncStep2, so the
 * client knows that the sync is finished.  There are two reasons for this more elaborated sync model: 1. This protocol can
 * easily be implemented on top of http and websockets. 2. The server should only reply to requests, and not initiate them.
 * Therefore it is necessary that the client initiates the sync.
 *
 * Construction of a message:
 * [messageType : varUint, message definition..]
 *
 * Note: A message does not include information about the room name. This must to be handled by the upper layer protocol!
 *
 * stringify[messageType] stringifies a message definition (messageType is already read from the bufffer)
 */

const messageYjsSyncStep1 = 0;
const messageYjsSyncStep2 = 1;
const messageYjsUpdate = 2;

/**
 * Create a sync step 1 message based on the state of the current shared document.
 *
 * @param {encoding.Encoder} encoder
 * @param {Y.Doc} doc
 */
const writeSyncStep1 = (encoder, doc) => {
  writeVarUint(encoder, messageYjsSyncStep1);
  const sv = Y__namespace.encodeStateVector(doc);
  writeVarUint8Array(encoder, sv);
};

/**
 * @param {encoding.Encoder} encoder
 * @param {Y.Doc} doc
 * @param {Uint8Array} [encodedStateVector]
 */
const writeSyncStep2 = (encoder, doc, encodedStateVector) => {
  writeVarUint(encoder, messageYjsSyncStep2);
  writeVarUint8Array(encoder, Y__namespace.encodeStateAsUpdate(doc, encodedStateVector));
};

/**
 * Read SyncStep1 message and reply with SyncStep2.
 *
 * @param {decoding.Decoder} decoder The reply to the received message
 * @param {encoding.Encoder} encoder The received message
 * @param {Y.Doc} doc
 */
const readSyncStep1 = (decoder, encoder, doc) =>
  writeSyncStep2(encoder, doc, readVarUint8Array(decoder));

/**
 * Read and apply Structs and then DeleteStore to a y instance.
 *
 * @param {decoding.Decoder} decoder
 * @param {Y.Doc} doc
 * @param {any} transactionOrigin
 */
const readSyncStep2 = (decoder, doc, transactionOrigin) => {
  try {
    Y__namespace.applyUpdate(doc, readVarUint8Array(decoder), transactionOrigin);
  } catch (error) {
    // This catches errors that are thrown by event handlers
    console.error('Caught error while handling a Yjs update', error);
  }
};

/**
 * @param {encoding.Encoder} encoder
 * @param {Uint8Array} update
 */
const writeUpdate = (encoder, update) => {
  writeVarUint(encoder, messageYjsUpdate);
  writeVarUint8Array(encoder, update);
};

/**
 * Read and apply Structs and then DeleteStore to a y instance.
 *
 * @param {decoding.Decoder} decoder
 * @param {Y.Doc} doc
 * @param {any} transactionOrigin
 */
const readUpdate = readSyncStep2;

/**
 * @param {decoding.Decoder} decoder A message received from another client
 * @param {encoding.Encoder} encoder The reply message. Does not need to be sent if empty.
 * @param {Y.Doc} doc
 * @param {any} transactionOrigin
 */
const readSyncMessage = (decoder, encoder, doc, transactionOrigin) => {
  const messageType = readVarUint(decoder);
  switch (messageType) {
    case messageYjsSyncStep1:
      readSyncStep1(decoder, encoder, doc);
      break
    case messageYjsSyncStep2:
      readSyncStep2(decoder, doc, transactionOrigin);
      break
    case messageYjsUpdate:
      readUpdate(decoder, doc, transactionOrigin);
      break
    default:
      throw new Error('Unknown message type')
  }
  return messageType
};

class OutgoingMessage {
    constructor() {
        this.encoder = createEncoder();
    }
    get(args) {
        return args.encoder;
    }
    toUint8Array() {
        return toUint8Array(this.encoder);
    }
}

class MessageReceiver {
    constructor(message) {
        this.broadcasted = false;
        this.message = message;
    }
    setBroadcasted(value) {
        this.broadcasted = value;
        return this;
    }
    apply(provider, emitSynced) {
        const { message } = this;
        const type = message.readVarUint();
        const emptyMessageLength = message.length();
        switch (type) {
            case exports.MessageType.Sync:
                this.applySyncMessage(provider, emitSynced);
                break;
            case exports.MessageType.Awareness:
                this.applyAwarenessMessage(provider);
                break;
            case exports.MessageType.Auth:
                this.applyAuthMessage(provider);
                break;
            case exports.MessageType.QueryAwareness:
                this.applyQueryAwarenessMessage(provider);
                break;
            case exports.MessageType.Stateless:
                provider.receiveStateless(readVarString(message.decoder));
                break;
            case exports.MessageType.SyncStatus:
                this.applySyncStatusMessage(provider, readVarInt(message.decoder) === 1);
                break;
            default:
                throw new Error(`Can’t apply message of unknown type: ${type}`);
        }
        // Reply
        if (message.length() > emptyMessageLength + 1) { // length of documentName (considered in emptyMessageLength plus length of yjs sync type, set in applySyncMessage)
            if (this.broadcasted) {
                // TODO: Some weird TypeScript error
                // @ts-ignore
                provider.broadcast(OutgoingMessage, { encoder: message.encoder });
            }
            else {
                // TODO: Some weird TypeScript error
                // @ts-ignore
                provider.send(OutgoingMessage, { encoder: message.encoder });
            }
        }
    }
    applySyncMessage(provider, emitSynced) {
        const { message } = this;
        message.writeVarUint(exports.MessageType.Sync);
        // Apply update
        const syncMessageType = readSyncMessage(message.decoder, message.encoder, provider.document, provider);
        // Synced once we receive Step2
        if (emitSynced && syncMessageType === messageYjsSyncStep2) {
            provider.synced = true;
        }
    }
    applySyncStatusMessage(provider, applied) {
        if (applied) {
            provider.decrementUnsyncedChanges();
        }
    }
    applyAwarenessMessage(provider) {
        if (!provider.awareness)
            return;
        const { message } = this;
        applyAwarenessUpdate(provider.awareness, message.readVarUint8Array(), provider);
    }
    applyAuthMessage(provider) {
        const { message } = this;
        common.readAuthMessage(message.decoder, provider.permissionDeniedHandler.bind(provider), provider.authenticatedHandler.bind(provider));
    }
    applyQueryAwarenessMessage(provider) {
        if (!provider.awareness)
            return;
        const { message } = this;
        message.writeVarUint(exports.MessageType.Awareness);
        message.writeVarUint8Array(encodeAwarenessUpdate(provider.awareness, Array.from(provider.awareness.getStates().keys())));
    }
}

class MessageSender {
    constructor(Message, args = {}) {
        this.message = new Message();
        this.encoder = this.message.get(args);
    }
    create() {
        return toUint8Array(this.encoder);
    }
    send(webSocket) {
        webSocket === null || webSocket === void 0 ? void 0 : webSocket.send(this.create());
    }
    broadcast(channel) {
        publish(channel, this.create());
    }
}

class AuthenticationMessage extends OutgoingMessage {
    constructor() {
        super(...arguments);
        this.type = exports.MessageType.Auth;
        this.description = 'Authentication';
    }
    get(args) {
        if (typeof args.token === 'undefined') {
            throw new Error('The authentication message requires `token` as an argument.');
        }
        writeVarString(this.encoder, args.documentName);
        writeVarUint(this.encoder, this.type);
        common.writeAuthentication(this.encoder, args.token);
        return this.encoder;
    }
}

class AwarenessMessage extends OutgoingMessage {
    constructor() {
        super(...arguments);
        this.type = exports.MessageType.Awareness;
        this.description = 'Awareness states update';
    }
    get(args) {
        if (typeof args.awareness === 'undefined') {
            throw new Error('The awareness message requires awareness as an argument');
        }
        if (typeof args.clients === 'undefined') {
            throw new Error('The awareness message requires clients as an argument');
        }
        writeVarString(this.encoder, args.documentName);
        writeVarUint(this.encoder, this.type);
        let awarenessUpdate;
        if (args.states === undefined) {
            awarenessUpdate = encodeAwarenessUpdate(args.awareness, args.clients);
        }
        else {
            awarenessUpdate = encodeAwarenessUpdate(args.awareness, args.clients, args.states);
        }
        writeVarUint8Array(this.encoder, awarenessUpdate);
        return this.encoder;
    }
}

class CloseMessage extends OutgoingMessage {
    constructor() {
        super(...arguments);
        this.type = exports.MessageType.CLOSE;
        this.description = 'Ask the server to close the connection';
    }
    get(args) {
        writeVarString(this.encoder, args.documentName);
        writeVarUint(this.encoder, this.type);
        return this.encoder;
    }
}

class QueryAwarenessMessage extends OutgoingMessage {
    constructor() {
        super(...arguments);
        this.type = exports.MessageType.QueryAwareness;
        this.description = 'Queries awareness states';
    }
    get(args) {
        writeVarString(this.encoder, args.documentName);
        writeVarUint(this.encoder, this.type);
        return this.encoder;
    }
}

class StatelessMessage extends OutgoingMessage {
    constructor() {
        super(...arguments);
        this.type = exports.MessageType.Stateless;
        this.description = 'A stateless message';
    }
    get(args) {
        var _a;
        writeVarString(this.encoder, args.documentName);
        writeVarUint(this.encoder, this.type);
        writeVarString(this.encoder, (_a = args.payload) !== null && _a !== void 0 ? _a : '');
        return this.encoder;
    }
}

class SyncStepOneMessage extends OutgoingMessage {
    constructor() {
        super(...arguments);
        this.type = exports.MessageType.Sync;
        this.description = 'First sync step';
    }
    get(args) {
        if (typeof args.document === 'undefined') {
            throw new Error('The sync step one message requires document as an argument');
        }
        writeVarString(this.encoder, args.documentName);
        writeVarUint(this.encoder, this.type);
        writeSyncStep1(this.encoder, args.document);
        return this.encoder;
    }
}

class SyncStepTwoMessage extends OutgoingMessage {
    constructor() {
        super(...arguments);
        this.type = exports.MessageType.Sync;
        this.description = 'Second sync step';
    }
    get(args) {
        if (typeof args.document === 'undefined') {
            throw new Error('The sync step two message requires document as an argument');
        }
        writeVarString(this.encoder, args.documentName);
        writeVarUint(this.encoder, this.type);
        writeSyncStep2(this.encoder, args.document);
        return this.encoder;
    }
}

class UpdateMessage extends OutgoingMessage {
    constructor() {
        super(...arguments);
        this.type = exports.MessageType.Sync;
        this.description = 'A document update';
    }
    get(args) {
        writeVarString(this.encoder, args.documentName);
        writeVarUint(this.encoder, this.type);
        writeUpdate(this.encoder, args.update);
        return this.encoder;
    }
}

class AwarenessError extends Error {
    constructor() {
        super(...arguments);
        this.code = 1001;
    }
}
class HocuspocusProvider extends EventEmitter {
    constructor(configuration) {
        var _a, _b, _c;
        super();
        this.configuration = {
            name: '',
            // @ts-ignore
            document: undefined,
            // @ts-ignore
            awareness: undefined,
            token: null,
            parameters: {},
            broadcast: true,
            forceSyncInterval: false,
            onAuthenticated: () => null,
            onAuthenticationFailed: () => null,
            onOpen: () => null,
            onConnect: () => null,
            onMessage: () => null,
            onOutgoingMessage: () => null,
            onStatus: () => null,
            onSynced: () => null,
            onDisconnect: () => null,
            onClose: () => null,
            onDestroy: () => null,
            onAwarenessUpdate: () => null,
            onAwarenessChange: () => null,
            onStateless: () => null,
            quiet: false,
            connect: true,
            preserveConnection: true,
        };
        this.subscribedToBroadcastChannel = false;
        this.isSynced = false;
        this.unsyncedChanges = 0;
        this.status = exports.WebSocketStatus.Disconnected;
        this.isAuthenticated = false;
        this.authorizedScope = undefined;
        this.mux = createMutex();
        this.intervals = {
            forceSync: null,
        };
        this.isConnected = true;
        this.boundDocumentUpdateHandler = this.documentUpdateHandler.bind(this);
        this.boundAwarenessUpdateHandler = this.awarenessUpdateHandler.bind(this);
        this.boundBroadcastChannelSubscriber = this.broadcastChannelSubscriber.bind(this);
        this.boundPageHide = this.pageHide.bind(this);
        this.boundOnOpen = this.onOpen.bind(this);
        this.boundOnClose = this.onClose.bind(this);
        this.boundOnStatus = this.onStatus.bind(this);
        this.forwardConnect = (e) => this.emit('connect', e);
        this.forwardOpen = (e) => this.emit('open', e);
        this.forwardClose = (e) => this.emit('close', e);
        this.forwardDisconnect = (e) => this.emit('disconnect', e);
        this.forwardDestroy = (e) => this.emit('destroy', e);
        this.setConfiguration(configuration);
        this.configuration.document = configuration.document ? configuration.document : new Y__namespace.Doc();
        this.configuration.awareness = configuration.awareness !== undefined ? configuration.awareness : new Awareness(this.document);
        this.on('open', this.configuration.onOpen);
        this.on('message', this.configuration.onMessage);
        this.on('outgoingMessage', this.configuration.onOutgoingMessage);
        this.on('synced', this.configuration.onSynced);
        this.on('destroy', this.configuration.onDestroy);
        this.on('awarenessUpdate', this.configuration.onAwarenessUpdate);
        this.on('awarenessChange', this.configuration.onAwarenessChange);
        this.on('stateless', this.configuration.onStateless);
        this.on('authenticated', this.configuration.onAuthenticated);
        this.on('authenticationFailed', this.configuration.onAuthenticationFailed);
        this.configuration.websocketProvider.on('connect', this.configuration.onConnect);
        this.configuration.websocketProvider.on('connect', this.forwardConnect);
        this.configuration.websocketProvider.on('open', this.boundOnOpen);
        this.configuration.websocketProvider.on('open', this.forwardOpen);
        this.configuration.websocketProvider.on('close', this.boundOnClose);
        this.configuration.websocketProvider.on('close', this.configuration.onClose);
        this.configuration.websocketProvider.on('close', this.forwardClose);
        this.configuration.websocketProvider.on('status', this.boundOnStatus);
        this.configuration.websocketProvider.on('disconnect', this.configuration.onDisconnect);
        this.configuration.websocketProvider.on('disconnect', this.forwardDisconnect);
        this.configuration.websocketProvider.on('destroy', this.configuration.onDestroy);
        this.configuration.websocketProvider.on('destroy', this.forwardDestroy);
        (_a = this.awareness) === null || _a === void 0 ? void 0 : _a.on('update', () => {
            this.emit('awarenessUpdate', { states: common.awarenessStatesToArray(this.awareness.getStates()) });
        });
        (_b = this.awareness) === null || _b === void 0 ? void 0 : _b.on('change', () => {
            this.emit('awarenessChange', { states: common.awarenessStatesToArray(this.awareness.getStates()) });
        });
        this.document.on('update', this.boundDocumentUpdateHandler);
        (_c = this.awareness) === null || _c === void 0 ? void 0 : _c.on('update', this.boundAwarenessUpdateHandler);
        this.registerEventListeners();
        if (this.configuration.forceSyncInterval
            && typeof this.configuration.forceSyncInterval === 'number') {
            this.intervals.forceSync = setInterval(this.forceSync.bind(this), this.configuration.forceSyncInterval);
        }
        this.configuration.websocketProvider.attach(this);
    }
    onStatus({ status }) {
        this.status = status;
        this.configuration.onStatus({ status });
        this.emit('status', { status });
    }
    setConfiguration(configuration = {}) {
        if (!configuration.websocketProvider && configuration.url) {
            const websocketProviderConfig = configuration;
            this.configuration.websocketProvider = new HocuspocusProviderWebsocket({
                url: websocketProviderConfig.url,
                connect: websocketProviderConfig.connect,
                parameters: websocketProviderConfig.parameters,
            });
        }
        this.configuration = { ...this.configuration, ...configuration };
    }
    get document() {
        return this.configuration.document;
    }
    get awareness() {
        return this.configuration.awareness;
    }
    get hasUnsyncedChanges() {
        return this.unsyncedChanges > 0;
    }
    resetUnsyncedChanges() {
        this.unsyncedChanges = 1;
        this.emit('unsyncedChanges', this.unsyncedChanges);
    }
    incrementUnsyncedChanges() {
        this.unsyncedChanges += 1;
        this.emit('unsyncedChanges', this.unsyncedChanges);
    }
    decrementUnsyncedChanges() {
        this.unsyncedChanges -= 1;
        if (this.unsyncedChanges === 0) {
            this.synced = true;
        }
        this.emit('unsyncedChanges', this.unsyncedChanges);
    }
    forceSync() {
        this.resetUnsyncedChanges();
        this.send(SyncStepOneMessage, { document: this.document, documentName: this.configuration.name });
    }
    pageHide() {
        if (this.awareness) {
            removeAwarenessStates(this.awareness, [this.document.clientID], 'page hide');
        }
    }
    registerEventListeners() {
        if (typeof window === 'undefined' || !('addEventListener' in window)) {
            return;
        }
        window.addEventListener('pagehide', this.boundPageHide);
    }
    sendStateless(payload) {
        this.send(StatelessMessage, { documentName: this.configuration.name, payload });
    }
    documentUpdateHandler(update, origin) {
        if (origin === this) {
            return;
        }
        this.incrementUnsyncedChanges();
        this.send(UpdateMessage, { update, documentName: this.configuration.name }, true);
    }
    awarenessUpdateHandler({ added, updated, removed }, origin) {
        const changedClients = added.concat(updated).concat(removed);
        this.send(AwarenessMessage, {
            awareness: this.awareness,
            clients: changedClients,
            documentName: this.configuration.name,
        }, true);
    }
    /**
     * Indicates whether a first handshake with the server has been established
     *
     * Note: this does not mean all updates from the client have been persisted to the backend. For this,
     * use `hasUnsyncedChanges`.
     */
    get synced() {
        return this.isSynced;
    }
    set synced(state) {
        if (this.isSynced === state) {
            return;
        }
        this.isSynced = state;
        this.emit('synced', { state });
        this.emit('sync', { state });
    }
    receiveStateless(payload) {
        this.emit('stateless', { payload });
    }
    get isAuthenticationRequired() {
        return !!this.configuration.token && !this.isAuthenticated;
    }
    // not needed, but provides backward compatibility with e.g. lexical/yjs
    async connect() {
        if (this.configuration.broadcast) {
            this.subscribeToBroadcastChannel();
        }
        this.configuration.websocketProvider.shouldConnect = true;
        return this.configuration.websocketProvider.attach(this);
    }
    disconnect() {
        this.disconnectBroadcastChannel();
        this.configuration.websocketProvider.detach(this);
        this.isConnected = false;
        if (!this.configuration.preserveConnection) {
            this.configuration.websocketProvider.disconnect();
        }
    }
    async onOpen(event) {
        this.isAuthenticated = false;
        this.isConnected = true;
        this.emit('open', { event });
        let token;
        try {
            token = await this.getToken();
        }
        catch (error) {
            this.permissionDeniedHandler(`Failed to get token: ${error}`);
            return;
        }
        if (this.isAuthenticationRequired) {
            this.send(AuthenticationMessage, {
                token,
                documentName: this.configuration.name,
            });
        }
        this.startSync();
    }
    async getToken() {
        if (typeof this.configuration.token === 'function') {
            const token = await this.configuration.token();
            return token;
        }
        return this.configuration.token;
    }
    startSync() {
        this.resetUnsyncedChanges();
        this.send(SyncStepOneMessage, { document: this.document, documentName: this.configuration.name });
        if (this.awareness && this.awareness.getLocalState() !== null) {
            this.send(AwarenessMessage, {
                awareness: this.awareness,
                clients: [this.document.clientID],
                documentName: this.configuration.name,
            });
        }
    }
    send(message, args, broadcast = false) {
        if (!this.isConnected) {
            return;
        }
        if (broadcast) {
            this.mux(() => { this.broadcast(message, args); });
        }
        const messageSender = new MessageSender(message, args);
        this.emit('outgoingMessage', { message: messageSender.message });
        messageSender.send(this.configuration.websocketProvider);
    }
    onMessage(event) {
        const message = new IncomingMessage(event.data);
        const documentName = message.readVarString();
        message.writeVarString(documentName);
        this.emit('message', { event, message: new IncomingMessage(event.data) });
        new MessageReceiver(message).apply(this, true);
    }
    onClose(event) {
        this.isAuthenticated = false;
        this.synced = false;
        // update awareness (all users except local left)
        if (this.awareness) {
            removeAwarenessStates(this.awareness, Array.from(this.awareness.getStates().keys()).filter(client => client !== this.document.clientID), this);
        }
    }
    destroy() {
        this.emit('destroy');
        if (this.intervals.forceSync) {
            clearInterval(this.intervals.forceSync);
        }
        if (this.awareness) {
            removeAwarenessStates(this.awareness, [this.document.clientID], 'provider destroy');
            this.awareness.off('update', this.boundAwarenessUpdateHandler);
            this.awareness.destroy();
        }
        this.document.off('update', this.boundDocumentUpdateHandler);
        this.removeAllListeners();
        this.configuration.websocketProvider.off('connect', this.configuration.onConnect);
        this.configuration.websocketProvider.off('connect', this.forwardConnect);
        this.configuration.websocketProvider.off('open', this.boundOnOpen);
        this.configuration.websocketProvider.off('open', this.forwardOpen);
        this.configuration.websocketProvider.off('close', this.boundOnClose);
        this.configuration.websocketProvider.off('close', this.configuration.onClose);
        this.configuration.websocketProvider.off('close', this.forwardClose);
        this.configuration.websocketProvider.off('status', this.boundOnStatus);
        this.configuration.websocketProvider.off('disconnect', this.configuration.onDisconnect);
        this.configuration.websocketProvider.off('disconnect', this.forwardDisconnect);
        this.configuration.websocketProvider.off('destroy', this.configuration.onDestroy);
        this.configuration.websocketProvider.off('destroy', this.forwardDestroy);
        this.send(CloseMessage, { documentName: this.configuration.name });
        this.disconnect();
        if (typeof window === 'undefined' || !('removeEventListener' in window)) {
            return;
        }
        window.removeEventListener('pagehide', this.boundPageHide);
    }
    permissionDeniedHandler(reason) {
        this.emit('authenticationFailed', { reason });
        this.isAuthenticated = false;
        this.disconnect();
        this.status = exports.WebSocketStatus.Disconnected;
    }
    authenticatedHandler(scope) {
        this.isAuthenticated = true;
        this.authorizedScope = scope;
        this.emit('authenticated');
    }
    get broadcastChannel() {
        return `${this.configuration.name}`;
    }
    broadcastChannelSubscriber(data) {
        this.mux(() => {
            const message = new IncomingMessage(data);
            const documentName = message.readVarString();
            message.writeVarString(documentName);
            new MessageReceiver(message)
                .setBroadcasted(true)
                .apply(this, false);
        });
    }
    subscribeToBroadcastChannel() {
        if (!this.subscribedToBroadcastChannel) {
            subscribe(this.broadcastChannel, this.boundBroadcastChannelSubscriber);
            this.subscribedToBroadcastChannel = true;
        }
        this.mux(() => {
            this.broadcast(SyncStepOneMessage, { document: this.document, documentName: this.configuration.name });
            this.broadcast(SyncStepTwoMessage, { document: this.document, documentName: this.configuration.name });
            this.broadcast(QueryAwarenessMessage, { document: this.document, documentName: this.configuration.name });
            if (this.awareness) {
                this.broadcast(AwarenessMessage, {
                    awareness: this.awareness,
                    clients: [this.document.clientID],
                    document: this.document,
                    documentName: this.configuration.name,
                });
            }
        });
    }
    disconnectBroadcastChannel() {
        // broadcast message with local awareness state set to null (indicating disconnect)
        if (this.awareness) {
            this.send(AwarenessMessage, {
                awareness: this.awareness,
                clients: [this.document.clientID],
                states: new Map(),
                documentName: this.configuration.name,
            }, true);
        }
        if (this.subscribedToBroadcastChannel) {
            unsubscribe(this.broadcastChannel, this.boundBroadcastChannelSubscriber);
            this.subscribedToBroadcastChannel = false;
        }
    }
    broadcast(Message, args) {
        if (!this.configuration.broadcast) {
            return;
        }
        if (!this.subscribedToBroadcastChannel) {
            return;
        }
        new MessageSender(Message, args).broadcast(this.broadcastChannel);
    }
    setAwarenessField(key, value) {
        if (!this.awareness) {
            throw new AwarenessError(`Cannot set awareness field "${key}" to ${JSON.stringify(value)}. You have disabled Awareness for this provider by explicitly passing awareness: null in the provider configuration.`);
        }
        this.awareness.setLocalStateField(key, value);
    }
}

/* eslint-env browser */
const getRandomValues = crypto.getRandomValues.bind(crypto);

/**
 * Isomorphic module for true random numbers / buffers / uuids.
 *
 * Attention: falls back to Math.random if the browser does not support crypto.
 *
 * @module random
 */

const uint32 = () => getRandomValues(new Uint32Array(1))[0];

// @ts-ignore
const uuidv4Template = [1e7] + -1e3 + -4e3 + -8e3 + -1e11;

/**
 * @return {string}
 */
const uuidv4 = () => uuidv4Template.replace(/[018]/g, /** @param {number} c */ c =>
  (c ^ uint32() & 15 >> c / 4).toString(16)
);

class TiptapCollabProviderWebsocket extends HocuspocusProviderWebsocket {
    constructor(configuration) {
        var _a;
        let url = (_a = configuration.baseUrl) !== null && _a !== void 0 ? _a : `wss://${configuration.appId}.collab.tiptap.cloud`;
        if (configuration.shardKey) {
            url += url.includes('?') ? '&' : '?';
            url += `shard=${configuration.shardKey}`;
        }
        super({ ...configuration, url });
    }
}

const defaultDeleteCommentOptions = {
    deleteContent: false,
    deleteThread: false,
};
const defaultGetThreadsOptions = {
    types: ['unarchived'],
};
const defaultDeleteThreadOptions = {
    deleteComments: false,
    force: false,
};
class TiptapCollabProvider extends HocuspocusProvider {
    constructor(configuration) {
        if (!configuration.websocketProvider) {
            configuration.websocketProvider = new TiptapCollabProviderWebsocket({ appId: configuration.appId, baseUrl: configuration.baseUrl });
        }
        if (!configuration.token) {
            configuration.token = 'notoken'; // need to send a token anyway (which will be ignored)
        }
        super(configuration);
        this.tiptapCollabConfigurationPrefix = '__tiptapcollab__';
        if (configuration.user) {
            this.userData = new Y__namespace.PermanentUserData(this.document, this.document.getMap('__tiptapcollab__users'));
            this.userData.setUserMapping(this.document, this.document.clientID, configuration.user);
        }
    }
    /**
     * note: this will only work if your server loaded @hocuspocus-pro/extension-history, or if you are on a Tiptap business plan.
     */
    createVersion(name) {
        return this.sendStateless(JSON.stringify({ action: 'version.create', name }));
    }
    /**
     * note: this will only work if your server loaded @hocuspocus-pro/extension-history, or if you are on a Tiptap business plan.
     */
    revertToVersion(targetVersion) {
        return this.sendStateless(JSON.stringify({ action: 'document.revert', version: targetVersion }));
    }
    /**
     * note: this will only work if your server loaded @hocuspocus-pro/extension-history, or if you are on a Tiptap business plan.
     *
     * The server will reply with a stateless message (THistoryVersionPreviewEvent)
     */
    previewVersion(targetVersion) {
        return this.sendStateless(JSON.stringify({ action: 'version.preview', version: targetVersion }));
    }
    /**
     * note: this will only work if your server loaded @hocuspocus-pro/extension-history, or if you are on a Tiptap business plan.
     */
    getVersions() {
        return this.configuration.document.getArray(`${this.tiptapCollabConfigurationPrefix}versions`).toArray();
    }
    watchVersions(callback) {
        return this.configuration.document.getArray('__tiptapcollab__versions').observe(callback);
    }
    unwatchVersions(callback) {
        return this.configuration.document.getArray('__tiptapcollab__versions').unobserve(callback);
    }
    isAutoVersioning() {
        return !!this.configuration.document.getMap(`${this.tiptapCollabConfigurationPrefix}config`).get('autoVersioning');
    }
    enableAutoVersioning() {
        return this.configuration.document.getMap(`${this.tiptapCollabConfigurationPrefix}config`).set('autoVersioning', 1);
    }
    disableAutoVersioning() {
        return this.configuration.document.getMap(`${this.tiptapCollabConfigurationPrefix}config`).set('autoVersioning', 0);
    }
    /**
     * Returns all users in the document as Y.Map objects
     * @returns An array of Y.Map objects
     */
    getYThreads() {
        return this.configuration.document.getArray(`${this.tiptapCollabConfigurationPrefix}threads`);
    }
    /**
     * Finds all threads in the document and returns them as JSON objects
     * @options Options to control the output of the threads (e.g. include deleted threads)
     * @returns An array of threads as JSON objects
     */
    getThreads(options) {
        const { types } = { ...defaultGetThreadsOptions, ...options };
        const threads = this.getYThreads().toJSON();
        if ((types === null || types === void 0 ? void 0 : types.includes('archived')) && (types === null || types === void 0 ? void 0 : types.includes('unarchived'))) {
            return threads;
        }
        return threads.filter(currentThead => {
            if ((types === null || types === void 0 ? void 0 : types.includes('archived')) && currentThead.deletedAt) {
                return true;
            }
            if ((types === null || types === void 0 ? void 0 : types.includes('unarchived')) && !currentThead.deletedAt) {
                return true;
            }
            return false;
        });
    }
    /**
     * Find the index of a thread by its id
     * @param id The thread id
     * @returns The index of the thread or null if not found
     */
    getThreadIndex(id) {
        let index = null;
        let i = 0;
        // eslint-disable-next-line no-restricted-syntax
        for (const thread of this.getThreads({ types: ['archived', 'unarchived'] })) {
            if (thread.id === id) {
                index = i;
                break;
            }
            i += 1;
        }
        return index;
    }
    /**
     * Gets a single thread by its id
     * @param id The thread id
     * @returns The thread as a JSON object or null if not found
     */
    getThread(id) {
        const index = this.getThreadIndex(id);
        if (index === null) {
            return null;
        }
        return this.getYThreads().get(index).toJSON();
    }
    /**
     * Gets a single thread by its id as a Y.Map object
     * @param id The thread id
     * @returns The thread as a Y.Map object or null if not found
     */
    getYThread(id) {
        const index = this.getThreadIndex(id);
        if (index === null) {
            return null;
        }
        return this.getYThreads().get(index);
    }
    /**
     * Create a new thread
     * @param data The thread data
     * @returns The created thread
     */
    createThread(data) {
        let createdThread = {};
        this.document.transact(() => {
            const thread = new Y__namespace.Map();
            thread.set('id', uuidv4());
            thread.set('createdAt', (new Date()).toISOString());
            thread.set('comments', new Y__namespace.Array());
            thread.set('deletedComments', new Y__namespace.Array());
            thread.set('deletedAt', null);
            this.getYThreads().push([thread]);
            createdThread = this.updateThread(String(thread.get('id')), data);
        });
        return createdThread;
    }
    /**
     * Update a specific thread
     * @param id The thread id
     * @param data New data for the thread
     * @returns The updated thread or null if the thread is not found
     */
    updateThread(id, data) {
        let updatedThread = {};
        this.document.transact(() => {
            const thread = this.getYThread(id);
            if (thread === null) {
                return null;
            }
            thread.set('updatedAt', (new Date()).toISOString());
            if (data.data) {
                thread.set('data', data.data);
            }
            if (data.resolvedAt || data.resolvedAt === null) {
                thread.set('resolvedAt', data.resolvedAt);
            }
            updatedThread = thread.toJSON();
        });
        return updatedThread;
    }
    /**
     * Handle the deletion of a thread. By default, the thread and it's comments are not deleted, but marked as deleted
     * via the `deletedAt` property. Forceful deletion can be enabled by setting the `force` option to `true`.
     *
     * If you only want to delete the comments of a thread, you can set the `deleteComments` option to `true`.
     * @param id The thread id
     * @param options A set of options that control how the thread is deleted
     * @returns The deleted thread or null if the thread is not found
     */
    deleteThread(id, options) {
        const { deleteComments, force } = { ...defaultDeleteThreadOptions, ...options };
        const index = this.getThreadIndex(id);
        if (index === null) {
            return null;
        }
        if (force) {
            this.getYThreads().delete(index, 1);
            return;
        }
        const thread = this.getYThreads().get(index);
        thread.set('deletedAt', (new Date()).toISOString());
        if (deleteComments) {
            thread.set('comments', new Y__namespace.Array());
            thread.set('deletedComments', new Y__namespace.Array());
        }
        return thread.toJSON();
    }
    /**
     * Tries to restore a deleted thread
     * @param id The thread id
     * @returns The restored thread or null if the thread is not found
     */
    restoreThread(id) {
        const index = this.getThreadIndex(id);
        if (index === null) {
            return null;
        }
        const thread = this.getYThreads().get(index);
        thread.set('deletedAt', null);
        return thread.toJSON();
    }
    /**
     * Returns comments from a thread, either deleted or not
     * @param threadId The thread id
     * @param includeDeleted If you want to include deleted comments, defaults to `false`
     * @returns The comments or null if the thread is not found
     */
    getThreadComments(threadId, includeDeleted) {
        var _a, _b, _c;
        const index = this.getThreadIndex(threadId);
        if (index === null) {
            return null;
        }
        const comments = !includeDeleted ? (_a = this.getThread(threadId)) === null || _a === void 0 ? void 0 : _a.comments : [...(((_b = this.getThread(threadId)) === null || _b === void 0 ? void 0 : _b.comments) || []), ...(((_c = this.getThread(threadId)) === null || _c === void 0 ? void 0 : _c.deletedComments) || [])].sort((a, b) => {
            return a.createdAt.localeCompare(b.createdAt);
        });
        return comments !== null && comments !== void 0 ? comments : [];
    }
    /**
     * Get a single comment from a specific thread
     * @param threadId The thread id
     * @param commentId The comment id
     * @param includeDeleted If you want to include deleted comments in the search
     * @returns The comment or null if not found
     */
    getThreadComment(threadId, commentId, includeDeleted) {
        var _a;
        const index = this.getThreadIndex(threadId);
        if (index === null) {
            return null;
        }
        const comments = this.getThreadComments(threadId, includeDeleted);
        return (_a = comments === null || comments === void 0 ? void 0 : comments.find(comment => comment.id === commentId)) !== null && _a !== void 0 ? _a : null;
    }
    /**
     * Adds a comment to a thread
     * @param threadId The thread id
     * @param data The comment data
     * @returns The updated thread or null if the thread is not found
     * @example addComment('123', { content: 'Hello world', data: { author: 'Maria Doe' } })
     */
    addComment(threadId, data) {
        let updatedThread = {};
        this.document.transact(() => {
            const thread = this.getYThread(threadId);
            if (thread === null)
                return null;
            const commentMap = new Y__namespace.Map();
            commentMap.set('id', uuidv4());
            commentMap.set('createdAt', (new Date()).toISOString());
            thread.get('comments').push([commentMap]);
            this.updateComment(threadId, String(commentMap.get('id')), data);
            updatedThread = thread.toJSON();
        });
        return updatedThread;
    }
    /**
     * Update a comment in a thread
     * @param threadId The thread id
     * @param commentId The comment id
     * @param data The new comment data
     * @returns The updated thread or null if the thread or comment is not found
     * @example updateComment('123', { content: 'The new content', data: { attachments: ['file1.jpg'] }})
     */
    updateComment(threadId, commentId, data) {
        let updatedThread = {};
        this.document.transact(() => {
            const thread = this.getYThread(threadId);
            if (thread === null)
                return null;
            let comment = null;
            // eslint-disable-next-line no-restricted-syntax
            for (const c of thread.get('comments')) {
                if (c.get('id') === commentId) {
                    comment = c;
                    break;
                }
            }
            if (comment === null)
                return null;
            comment.set('updatedAt', (new Date()).toISOString());
            if (data.data) {
                comment.set('data', data.data);
            }
            if (data.content) {
                comment.set('content', data.content);
            }
            updatedThread = thread.toJSON();
        });
        return updatedThread;
    }
    /**
     * Deletes a comment from a thread
     * @param threadId The thread id
     * @param commentId The comment id
     * @param options A set of options that control how the comment is deleted
     * @returns The updated thread or null if the thread or comment is not found
     */
    deleteComment(threadId, commentId, options) {
        const { deleteContent, deleteThread } = { ...defaultDeleteCommentOptions, ...options };
        const thread = this.getYThread(threadId);
        if (thread === null)
            return null;
        let commentIndex = 0;
        // eslint-disable-next-line no-restricted-syntax
        for (const c of thread.get('comments')) {
            if (c.get('id') === commentId) {
                break;
            }
            commentIndex += 1;
        }
        // if the first comment of a thread is deleted we also
        // delete the thread itself as the source comment is gone
        if (commentIndex === 0 && (deleteThread || this.configuration.deleteThreadOnFirstCommentDelete)) {
            this.deleteThread(threadId);
            return;
        }
        const comment = thread.get('comments').get(commentIndex);
        const newComment = new Y__namespace.Map();
        newComment.set('id', comment.get('id'));
        newComment.set('createdAt', comment.get('createdAt'));
        newComment.set('updatedAt', (new Date()).toISOString());
        newComment.set('deletedAt', (new Date()).toISOString());
        newComment.set('data', comment.get('data'));
        newComment.set('content', deleteContent ? null : comment.get('content'));
        if (!thread.get('deletedComments')) {
            thread.set('deletedComments', new Y__namespace.Array());
        }
        thread.get('deletedComments').push([newComment]);
        thread.get('comments').delete(commentIndex);
        return thread.toJSON();
    }
    /**
     * Start watching threads for changes
     * @param callback The callback function to be called when a thread changes
     */
    watchThreads(callback) {
        this.getYThreads().observeDeep(callback);
    }
    /**
     * Stop watching threads for changes
     * @param callback The callback function to be removed
     */
    unwatchThreads(callback) {
        this.getYThreads().unobserveDeep(callback);
    }
}

exports.AwarenessError = AwarenessError;
exports.HocuspocusProvider = HocuspocusProvider;
exports.HocuspocusProviderWebsocket = HocuspocusProviderWebsocket;
exports.TiptapCollabProvider = TiptapCollabProvider;
exports.TiptapCollabProviderWebsocket = TiptapCollabProviderWebsocket;
//# sourceMappingURL=hocuspocus-provider.cjs.map
