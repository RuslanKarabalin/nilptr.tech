// Admin session handling: cookie forwarding, Set-Cookie relay and the
// server-side token refresh. It does not depend on SvelteKit so it can be
// unit tested with plain objects.

import type { AdminUser } from '$lib/types';

export const ACCESS_COOKIE = 'access_token';
export const REFRESH_COOKIE = 'refresh_token';

/** Default cookie paths from docs/API.md, used when the backend omits Path. */
const DEFAULT_PATHS: Record<string, string> = {
	[ACCESS_COOKIE]: '/',
	[REFRESH_COOKIE]: '/admin'
};

export type SameSite = 'strict' | 'lax' | 'none';

export interface CookieOptions {
	path: string;
	domain?: string;
	maxAge?: number;
	expires?: Date;
	httpOnly?: boolean;
	secure?: boolean;
	sameSite?: SameSite;
	encode?: (value: string) => string;
	decode?: (value: string) => string;
}

/** Subset of SvelteKit's Cookies used here. */
export interface CookieJar {
	get(name: string, opts?: { decode?: (value: string) => string }): string | undefined;
	set(name: string, value: string, opts: CookieOptions): void;
	delete(name: string, opts: CookieOptions): void;
}

export interface ClientInfo {
	userAgent: string;
	clientAddress: string;
}

export interface ParsedCookie {
	name: string;
	value: string;
	options: CookieOptions;
	/** True when the header removes the cookie (Max-Age <= 0 or Expires in the past). */
	expired: boolean;
}

const identity = (value: string) => value;

/** Parses one Set-Cookie header value. Returns null for malformed input. */
export function parseSetCookie(header: string, now = Date.now()): ParsedCookie | null {
	const [pair, ...attrs] = header.split(';');
	const eq = pair.indexOf('=');
	if (eq <= 0) return null;
	const name = pair.slice(0, eq).trim();
	let value = pair.slice(eq + 1).trim();
	if (value.length >= 2 && value.startsWith('"') && value.endsWith('"')) {
		value = value.slice(1, -1);
	}
	if (!name) return null;

	const options: CookieOptions = { path: DEFAULT_PATHS[name] ?? '/' };
	let expired = false;

	for (const attr of attrs) {
		const idx = attr.indexOf('=');
		const key = (idx === -1 ? attr : attr.slice(0, idx)).trim().toLowerCase();
		const val = idx === -1 ? '' : attr.slice(idx + 1).trim();
		switch (key) {
			case 'path':
				if (val.startsWith('/')) options.path = val;
				break;
			case 'domain':
				if (val) options.domain = val;
				break;
			case 'max-age': {
				const n = Number.parseInt(val, 10);
				if (!Number.isNaN(n)) {
					options.maxAge = n;
					if (n <= 0) expired = true;
				}
				break;
			}
			case 'expires': {
				const date = new Date(val);
				if (!Number.isNaN(date.getTime())) {
					options.expires = date;
					if (options.maxAge === undefined && date.getTime() <= now) expired = true;
				}
				break;
			}
			case 'httponly':
				options.httpOnly = true;
				break;
			case 'secure':
				options.secure = true;
				break;
			case 'samesite': {
				const v = val.toLowerCase();
				if (v === 'strict' || v === 'lax' || v === 'none') options.sameSite = v;
				break;
			}
		}
	}
	// SvelteKit defaults these to true, keep exactly what the backend sent.
	options.httpOnly ??= false;
	options.secure ??= false;
	options.sameSite ??= 'lax';

	return { name, value, options, expired };
}

/**
 * Copies Set-Cookie headers of a backend response to the browser response.
 * Values are written verbatim (no URL encoding) so the backend receives
 * exactly the token it issued.
 */
export function relaySetCookies(headers: Headers, cookies: CookieJar): ParsedCookie[] {
	const relayed: ParsedCookie[] = [];
	for (const header of headers.getSetCookie()) {
		const cookie = parseSetCookie(header);
		if (!cookie) continue;
		if (cookie.expired || cookie.value === '') {
			cookies.delete(cookie.name, { ...cookie.options, maxAge: 0, expires: undefined });
		} else {
			cookies.set(cookie.name, cookie.value, { ...cookie.options, encode: identity });
		}
		relayed.push(cookie);
	}
	return relayed;
}

/** Reads a cookie without URL decoding. Empty values count as missing. */
export function readCookie(cookies: CookieJar, name: string): string | undefined {
	const value = cookies.get(name, { decode: identity });
	return value ? value : undefined;
}

/**
 * True if the JWT is missing an exp claim or expires within `skewSeconds`.
 * The signature is not checked, that is the backend's job.
 */
export function isTokenExpired(token: string, nowSeconds = Date.now() / 1000, skewSeconds = 10) {
	const parts = token.split('.');
	if (parts.length !== 3) return true;
	try {
		const payload = JSON.parse(Buffer.from(parts[1], 'base64url').toString('utf8'));
		if (typeof payload.exp !== 'number') return true;
		return payload.exp <= nowSeconds + skewSeconds;
	} catch {
		return true;
	}
}

/** Headers for a backend call made on behalf of the browser. */
export function forwardHeaders(cookies: CookieJar, client: ClientInfo): Record<string, string> {
	const headers: Record<string, string> = {};
	const pairs: string[] = [];
	for (const name of [ACCESS_COOKIE, REFRESH_COOKIE]) {
		const value = readCookie(cookies, name);
		if (value) pairs.push(`${name}=${value}`);
	}
	if (pairs.length) headers.cookie = pairs.join('; ');
	if (client.userAgent) headers['user-agent'] = client.userAgent;
	if (client.clientAddress) headers['x-forwarded-for'] = client.clientAddress;
	return headers;
}

export interface SessionDeps {
	cookies: CookieJar;
	client: ClientInfo;
	fetch: typeof fetch;
	baseUrl: string;
}

/** Calls POST /api/admin/refresh and relays the cookies. Returns true on success. */
export async function refreshSession(deps: SessionDeps): Promise<boolean> {
	const res = await deps.fetch(`${deps.baseUrl}/api/admin/refresh`, {
		method: 'POST',
		headers: forwardHeaders(deps.cookies, deps.client)
	});
	relaySetCookies(res.headers, deps.cookies);
	await res.body?.cancel();
	return res.ok;
}

async function fetchMe(deps: SessionDeps): Promise<AdminUser | 'unauthorized'> {
	const res = await deps.fetch(`${deps.baseUrl}/api/admin/me`, {
		headers: { accept: 'application/json', ...forwardHeaders(deps.cookies, deps.client) }
	});
	if (res.status === 401) {
		await res.body?.cancel();
		return 'unauthorized';
	}
	if (!res.ok) {
		await res.body?.cancel();
		throw new Error(`GET /api/admin/me failed with ${res.status}`);
	}
	return (await res.json()) as AdminUser;
}

/**
 * Resolves the current admin. Refreshes the tokens at most once per call:
 * up front when the access token is missing or expired, or after a 401 from
 * /api/admin/me (for example when the token was signed with a rotated key).
 */
export async function resolveAdmin(deps: SessionDeps): Promise<AdminUser | null> {
	let refreshed = false;
	const hasRefresh = () => readCookie(deps.cookies, REFRESH_COOKIE) !== undefined;

	const access = readCookie(deps.cookies, ACCESS_COOKIE);
	if ((!access || isTokenExpired(access)) && hasRefresh()) {
		refreshed = true;
		if (!(await refreshSession(deps))) return null;
	}

	if (!readCookie(deps.cookies, ACCESS_COOKIE)) return null;

	const me = await fetchMe(deps);
	if (me !== 'unauthorized') return me;

	if (refreshed || !hasRefresh()) return null;
	if (!(await refreshSession(deps))) return null;
	const retry = await fetchMe(deps);
	return retry === 'unauthorized' ? null : retry;
}
