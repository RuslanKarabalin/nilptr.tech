import { describe, expect, it, vi } from 'vitest';
import {
	forwardHeaders,
	isTokenExpired,
	parseSetCookie,
	relaySetCookies,
	resolveAdmin,
	type CookieJar,
	type CookieOptions
} from './session';

interface Stored {
	value: string;
	options: CookieOptions;
}

/** Minimal stand-in for SvelteKit's cookies: request cookies plus cookies set during the request. */
function jar(initial: Record<string, string> = {}) {
	const request = new Map(Object.entries(initial));
	const written = new Map<string, Stored>();
	const cookies: CookieJar = {
		get(name) {
			if (written.has(name)) return written.get(name)!.value;
			return request.get(name);
		},
		set(name, value, options) {
			written.set(name, { value, options });
		},
		delete(name, options) {
			written.set(name, { value: '', options: { ...options, maxAge: 0 } });
		}
	};
	return { cookies, written };
}

function jwt(exp: number) {
	const enc = (v: object) => Buffer.from(JSON.stringify(v)).toString('base64url');
	return `${enc({ alg: 'HS256', typ: 'JWT' })}.${enc({ sub: 'u1', sid: 's1', exp })}.sig`;
}

const now = () => Math.floor(Date.now() / 1000);
const client = { userAgent: 'Mozilla/5.0 Test', clientAddress: '203.0.113.7' };
const baseUrl = 'http://backend.test';

const ACCESS_SET = (token: string) =>
	`access_token=${token}; Path=/; Max-Age=900; HttpOnly; Secure; SameSite=Strict`;
const REFRESH_SET = (token: string) =>
	`refresh_token=${token}; Path=/admin; Max-Age=2592000; HttpOnly; Secure; SameSite=Strict`;

function withCookies(status: number, setCookies: string[], body?: unknown) {
	const headers = new Headers();
	for (const c of setCookies) headers.append('set-cookie', c);
	if (body !== undefined) headers.set('content-type', 'application/json');
	return new Response(body === undefined ? null : JSON.stringify(body), { status, headers });
}

describe('parseSetCookie', () => {
	it('parses all attributes', () => {
		const c = parseSetCookie(REFRESH_SET('ab+c/d=='));
		expect(c).toEqual({
			name: 'refresh_token',
			value: 'ab+c/d==',
			expired: false,
			options: {
				path: '/admin',
				maxAge: 2592000,
				httpOnly: true,
				secure: true,
				sameSite: 'strict'
			}
		});
	});

	it('keeps insecure cookies insecure and defaults the path by name', () => {
		const c = parseSetCookie('refresh_token=x; HttpOnly; SameSite=Lax');
		expect(c?.options).toEqual({ path: '/admin', httpOnly: true, secure: false, sameSite: 'lax' });
	});

	it('detects clearing cookies', () => {
		expect(parseSetCookie('access_token=; Path=/; Max-Age=0')?.expired).toBe(true);
		expect(
			parseSetCookie('access_token=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT')?.expired
		).toBe(true);
	});

	it('rejects malformed headers', () => {
		expect(parseSetCookie('garbage')).toBeNull();
		expect(parseSetCookie('=value')).toBeNull();
	});
});

describe('relaySetCookies', () => {
	it('sets cookies verbatim with their paths and deletes cleared ones', () => {
		const { cookies, written } = jar();
		const headers = new Headers();
		headers.append('set-cookie', ACCESS_SET('a.b.c'));
		headers.append('set-cookie', REFRESH_SET('r+/='));
		headers.append('set-cookie', 'other=; Path=/; Max-Age=0');
		relaySetCookies(headers, cookies);

		const access = written.get('access_token')!;
		expect(access.value).toBe('a.b.c');
		expect(access.options).toMatchObject({ path: '/', maxAge: 900, httpOnly: true, secure: true });

		const refresh = written.get('refresh_token')!;
		expect(refresh.value).toBe('r+/=');
		expect(refresh.options.path).toBe('/admin');
		expect(refresh.options.encode?.('r+/=')).toBe('r+/=');

		expect(written.get('other')).toMatchObject({ value: '', options: { path: '/', maxAge: 0 } });
	});
});

describe('isTokenExpired', () => {
	it('checks exp with a skew', () => {
		expect(isTokenExpired(jwt(now() + 600))).toBe(false);
		expect(isTokenExpired(jwt(now() + 5))).toBe(true);
		expect(isTokenExpired(jwt(now() - 1))).toBe(true);
		expect(isTokenExpired('not-a-jwt')).toBe(true);
	});
});

describe('forwardHeaders', () => {
	it('forwards only the auth cookies, user agent and client address', () => {
		const { cookies } = jar({ access_token: 'a', refresh_token: 'r', theme: 'dark' });
		expect(forwardHeaders(cookies, client)).toEqual({
			cookie: 'access_token=a; refresh_token=r',
			'user-agent': 'Mozilla/5.0 Test',
			'x-forwarded-for': '203.0.113.7'
		});
	});
});

describe('resolveAdmin', () => {
	const user = { id: 'u1', login: 'admin' };

	it('uses a valid access token without refreshing', async () => {
		const { cookies } = jar({ access_token: jwt(now() + 600), refresh_token: 'r1' });
		const fetchMock = vi.fn(async () => Response.json(user));
		const result = await resolveAdmin({ cookies, client, fetch: fetchMock, baseUrl });
		expect(result).toEqual(user);
		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(fetchMock.mock.calls[0]).toEqual([
			`${baseUrl}/api/admin/me`,
			expect.objectContaining({
				headers: expect.objectContaining({ 'x-forwarded-for': '203.0.113.7' })
			})
		]);
	});

	it('refreshes once when the access token is missing and relays new cookies', async () => {
		const { cookies, written } = jar({ refresh_token: 'r1' });
		const fresh = jwt(now() + 900);
		const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			const headers = init?.headers as Record<string, string>;
			if (url.endsWith('/api/admin/refresh')) {
				expect(init?.method).toBe('POST');
				expect(headers.cookie).toBe('refresh_token=r1');
				expect(headers['user-agent']).toBe(client.userAgent);
				return withCookies(204, [ACCESS_SET(fresh), REFRESH_SET('r2')]);
			}
			expect(headers.cookie).toBe(`access_token=${fresh}; refresh_token=r2`);
			return Response.json(user);
		});

		const result = await resolveAdmin({ cookies, client, fetch: fetchMock, baseUrl });
		expect(result).toEqual(user);
		expect(fetchMock).toHaveBeenCalledTimes(2);
		expect(written.get('access_token')?.options.path).toBe('/');
		expect(written.get('refresh_token')).toMatchObject({
			value: 'r2',
			options: { path: '/admin' }
		});
	});

	it('refreshes when the access token is expired', async () => {
		const { cookies } = jar({ access_token: jwt(now() - 60), refresh_token: 'r1' });
		const fetchMock = vi.fn(async (input: RequestInfo | URL) =>
			String(input).endsWith('/refresh')
				? withCookies(204, [ACCESS_SET(jwt(now() + 900)), REFRESH_SET('r2')])
				: Response.json(user)
		);
		expect(await resolveAdmin({ cookies, client, fetch: fetchMock, baseUrl })).toEqual(user);
		expect(fetchMock).toHaveBeenCalledTimes(2);
	});

	it('clears cookies and returns null when the refresh fails', async () => {
		const { cookies, written } = jar({ refresh_token: 'stolen' });
		const fetchMock = vi.fn(async () =>
			withCookies(401, [
				'access_token=; Path=/; Max-Age=0',
				'refresh_token=; Path=/admin; Max-Age=0'
			])
		);
		expect(await resolveAdmin({ cookies, client, fetch: fetchMock, baseUrl })).toBeNull();
		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(written.get('refresh_token')).toMatchObject({
			value: '',
			options: { path: '/admin', maxAge: 0 }
		});
	});

	it('refreshes after a 401 from /me, but never twice', async () => {
		const { cookies } = jar({ access_token: jwt(now() + 600), refresh_token: 'r1' });
		const fetchMock = vi.fn(async (input: RequestInfo | URL) =>
			String(input).endsWith('/refresh')
				? withCookies(204, [ACCESS_SET(jwt(now() + 900)), REFRESH_SET('r2')])
				: Response.json({ error: 'unauthorized' }, { status: 401 })
		);
		expect(await resolveAdmin({ cookies, client, fetch: fetchMock, baseUrl })).toBeNull();
		const refreshCalls = fetchMock.mock.calls.filter(([u]) => String(u).endsWith('/refresh'));
		expect(refreshCalls).toHaveLength(1);
		expect(fetchMock).toHaveBeenCalledTimes(3);
	});

	it('returns null without any backend call when there are no cookies', async () => {
		const { cookies } = jar();
		const fetchMock = vi.fn();
		expect(await resolveAdmin({ cookies, client, fetch: fetchMock, baseUrl })).toBeNull();
		expect(fetchMock).not.toHaveBeenCalled();
	});
});
