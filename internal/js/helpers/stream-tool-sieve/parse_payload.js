'use strict';

const TOOLS_WRAPPER_PATTERN = /<tools\b[^>]*>([\s\S]*?)<\/tools>/gi;
const TOOL_CALL_MARKUP_BLOCK_PATTERN = /<(?:[a-z0-9_:-]+:)?tool_call\b[^>]*>([\s\S]*?)<\/(?:[a-z0-9_:-]+:)?tool_call>/gi;
const TOOL_CALL_CANONICAL_BODY_PATTERN = /^\s*<(?:[a-z0-9_:-]+:)?tool_name\b[^>]*>([\s\S]*?)<\/(?:[a-z0-9_:-]+:)?tool_name>\s*<(?:[a-z0-9_:-]+:)?param\b[^>]*>([\s\S]*?)<\/(?:[a-z0-9_:-]+:)?param>\s*$/i;
const TOOL_CALL_MARKUP_KV_PATTERN = /<(?:[a-z0-9_:-]+:)?([a-z0-9_.-]+)\b[^>]*>([\s\S]*?)<\/(?:[a-z0-9_:-]+:)?\1>/gi;
const CDATA_PATTERN = /^<!\[CDATA\[([\s\S]*?)]]>$/i;

const {
  toStringSafe,
} = require('./state');

function stripFencedCodeBlocks(text) {
  const t = typeof text === 'string' ? text : '';
  if (!t) {
    return '';
  }
  return t.replace(/```[\s\S]*?```/g, ' ');
}

function parseMarkupToolCalls(text) {
  const raw = toStringSafe(text).trim();
  if (!raw) {
    return [];
  }
  const out = [];
  for (const wrapper of raw.matchAll(TOOLS_WRAPPER_PATTERN)) {
    const body = toStringSafe(wrapper[1]);
    for (const block of body.matchAll(TOOL_CALL_MARKUP_BLOCK_PATTERN)) {
      const parsed = parseMarkupSingleToolCall(toStringSafe(block[1]).trim());
      if (parsed) {
        out.push(parsed);
      }
    }
  }
  return out;
}

function parseMarkupSingleToolCall(inner) {
  // Try inline JSON parse for the inner content.
  if (inner) {
    try {
      const decoded = JSON.parse(inner);
      if (decoded && typeof decoded === 'object' && !Array.isArray(decoded) && decoded.name) {
        return {
          name: toStringSafe(decoded.name),
          input: decoded.input && typeof decoded.input === 'object' && !Array.isArray(decoded.input) ? decoded.input : {},
        };
      }
    } catch (_err) {
      // Not JSON, continue with markup parsing.
    }
  }

  const match = inner.match(TOOL_CALL_CANONICAL_BODY_PATTERN);
  if (!match || match.length < 3) {
    return null;
  }

  const name = extractRawTagValue(match[1]).trim();
  if (!name) {
    return null;
  }

  const input = parseMarkupInput(match[2]);
  return { name, input };
}

function parseMarkupInput(raw) {
  const s = toStringSafe(raw).trim();
  if (!s) {
    return {};
  }
  // Prioritize XML-style KV tags (e.g., <arg>val</arg>)
  const kv = parseMarkupKVObject(s);
  if (Object.keys(kv).length > 0) {
    return kv;
  }

  // Fallback to JSON parsing
  const parsed = parseToolCallInput(s);
  if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
    if (Object.keys(parsed).length > 0) {
      return parsed;
    }
  }

  return { _raw: extractRawTagValue(s) };
}

function parseMarkupKVObject(text) {
  const raw = toStringSafe(text).trim();
  if (!raw) {
    return {};
  }
  const out = {};
  for (const m of raw.matchAll(TOOL_CALL_MARKUP_KV_PATTERN)) {
    const key = toStringSafe(m[1]).trim();
    if (!key) {
      continue;
    }
    const value = parseMarkupValue(m[2]);
    if (value === undefined || value === null) {
      continue;
    }
    appendMarkupValue(out, key, value);
  }
  return out;
}

function parseMarkupValue(raw) {
  const s = toStringSafe(extractRawTagValue(raw)).trim();
  if (!s) {
    return '';
  }

  if (s.includes('<') && s.includes('>')) {
    const nested = parseMarkupInput(s);
    if (nested && typeof nested === 'object' && !Array.isArray(nested)) {
      if (isOnlyRawValue(nested)) {
        return toStringSafe(nested._raw);
      }
      return nested;
    }
  }

  try {
    return JSON.parse(s);
  } catch (_err) {
    return s;
  }
}

function extractRawTagValue(inner) {
  const s = toStringSafe(inner).trim();
  if (!s) {
    return '';
  }

  // 1. Check for CDATA
  const cdataMatch = s.match(CDATA_PATTERN);
  if (cdataMatch && cdataMatch[1] !== undefined) {
    return cdataMatch[1];
  }

  // 2. Fallback to unescaping standard HTML entities
  // Note: we avoid broad tag stripping here to preserve user content (like < symbols in code)
  return unescapeHtml(inner);
}

function unescapeHtml(safe) {
  if (!safe) return '';
  return safe.replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#039;/g, "'")
    .replace(/&#x27;/g, "'");
}

function parseToolCallInput(v) {
  if (v == null) {
    return {};
  }
  if (typeof v === 'string') {
    const raw = toStringSafe(v);
    if (!raw) {
      return {};
    }
    try {
      const parsed = JSON.parse(raw);
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return parsed;
      }
      return { _raw: raw };
    } catch (_err) {
      return { _raw: raw };
    }
  }
  if (typeof v === 'object' && !Array.isArray(v)) {
    return v;
  }
  try {
    const parsed = JSON.parse(JSON.stringify(v));
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed;
    }
  } catch (_err) {
    return {};
  }
  return {};
}

function appendMarkupValue(out, key, value) {
  if (Object.prototype.hasOwnProperty.call(out, key)) {
    const current = out[key];
    if (Array.isArray(current)) {
      current.push(value);
      return;
    }
    out[key] = [current, value];
    return;
  }
  out[key] = value;
}

function isOnlyRawValue(obj) {
  if (!obj || typeof obj !== 'object' || Array.isArray(obj)) {
    return false;
  }
  const keys = Object.keys(obj);
  return keys.length === 1 && keys[0] === '_raw';
}

module.exports = {
  stripFencedCodeBlocks,
  parseMarkupToolCalls,
};
