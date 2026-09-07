const shapeLimits = { depth: 4, string: 4096, stack: 2048, entries: 256 };

function shape(value, depth) {
  if (value === null || value === undefined) {
    return null;
  }
  const kind = typeof value;
  if (kind === "string") {
    return value.length > shapeLimits.string ? value.slice(0, shapeLimits.string) : value;
  }
  if (kind === "number" || kind === "boolean") {
    return value;
  }
  if (kind === "bigint" || kind === "symbol") {
    return String(value);
  }
  if (value instanceof Error) {
    return {
      name: value.name,
      message: String(value.message ?? "").slice(0, shapeLimits.string),
      code: value.code ?? null,
      stack: String(value.stack ?? "").slice(0, shapeLimits.stack)
    };
  }
  if (kind === "object") {
    if (depth >= shapeLimits.depth) {
      return null;
    }
    if (Array.isArray(value)) {
      return value.slice(0, shapeLimits.entries).map((entry) => shape(entry, depth + 1));
    }
    const out = {};
    for (const [key, entry] of Object.entries(value).slice(0, shapeLimits.entries)) {
      out[key] = shape(entry, depth + 1);
    }
    return out;
  }
  return null;
}

export default async function* machineryReporter(source) {
  for await (const event of source) {
    let line;
    try {
      line = JSON.stringify({ t: event.type, d: shape(event.data, 0) });
    } catch {
      line = JSON.stringify({ t: String(event.type) });
    }
    yield line + "\n";
  }
  yield JSON.stringify({ t: "machinery:reporter:end" }) + "\n";
}
