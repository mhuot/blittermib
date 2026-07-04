import { type APIRequestContext } from '@playwright/test';

export interface ClientTiming {
  /** DEBOUNCE_MS from palette.js — the typeahead's trailing debounce. */
  debounceMs: number;
  /** MIN_QUERY_LEN from palette.js — the client-side gate threshold. */
  minQueryLen: number;
}

// clientTiming reads the typeahead's tuning constants straight from the
// served palette.js, so specs derive their waits from the real values
// instead of hard-coding magic numbers. A raised debounce would
// otherwise silently turn the negative gate assertion into a false pass;
// reading the source of truth keeps the timing envelope self-consistent
// (and fails loudly if the constants are renamed).
export async function clientTiming(
  request: APIRequestContext,
  baseURL: string,
): Promise<ClientTiming> {
  const res = await request.get(`${baseURL}/static/palette.js`);
  if (!res.ok()) throw new Error(`fetch palette.js: HTTP ${res.status()}`);
  const js = await res.text();
  const readInt = (re: RegExp, name: string): number => {
    const m = re.exec(js);
    if (!m) throw new Error(`could not read ${name} from palette.js`);
    return Number(m[1]);
  };
  return {
    debounceMs: readInt(/DEBOUNCE_MS\s*=\s*(\d+)/, 'DEBOUNCE_MS'),
    minQueryLen: readInt(/MIN_QUERY_LEN\s*=\s*(\d+)/, 'MIN_QUERY_LEN'),
  };
}
