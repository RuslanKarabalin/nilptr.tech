// Admin backend calls made by SvelteKit on behalf of the logged in browser.

import { error, fail, isHttpError, isRedirect, redirect, type RequestEvent } from '@sveltejs/kit';
import { ApiError, backendFetch, backendJson, type RequestOptions } from './backend';
import { forwardHeaders, type ClientInfo } from './session';

type Query = RequestOptions['query'];

export function clientInfo(event: RequestEvent): ClientInfo {
	let clientAddress = '';
	try {
		clientAddress = event.getClientAddress();
	} catch {
		// not available in some environments (for example during build)
	}
	return { userAgent: event.request.headers.get('user-agent') ?? '', clientAddress };
}

export function adminHeaders(event: RequestEvent): Record<string, string> {
	return forwardHeaders(event.cookies, clientInfo(event));
}

export function adminApi(event: RequestEvent) {
	const call = <T>(method: string, path: string, body?: unknown, query?: Query) =>
		backendJson<T>(`/api/admin${path}`, {
			method,
			query,
			headers: adminHeaders(event),
			body: body === undefined ? undefined : JSON.stringify(body)
		});

	return {
		get: <T>(path: string, query?: Query) => call<T>('GET', path, undefined, query),
		post: <T>(path: string, body?: unknown) => call<T>('POST', path, body),
		put: <T>(path: string, body?: unknown) => call<T>('PUT', path, body),
		del: (path: string) => call<void>('DELETE', path),
		/** Raw request with the session headers, for streaming bodies. */
		raw: (path: string, init: RequestOptions) =>
			backendFetch(`/api/admin${path}`, {
				...init,
				headers: { ...adminHeaders(event), ...Object.fromEntries(new Headers(init.headers)) }
			})
	};
}

/** Converts a failed backend call in a load function into a SvelteKit error. */
export function loadFailed(err: unknown): never {
	if (isRedirect(err) || isHttpError(err)) throw err;
	if (err instanceof ApiError) {
		if (err.status === 401) redirect(303, '/admin/login');
		error(err.status >= 400 && err.status < 600 ? err.status : 502, err.message);
	}
	throw err;
}

/** Converts a failed backend call in a form action into `fail()`. */
// eslint-disable-next-line @typescript-eslint/no-empty-object-type -- no extra fields by default
export function actionFailed<T extends object = {}>(err: unknown, extra?: T) {
	if (isRedirect(err) || isHttpError(err)) throw err;
	if (err instanceof ApiError) {
		if (err.status === 401) redirect(303, '/admin/login');
		return fail(err.status >= 400 && err.status < 600 ? err.status : 502, {
			...(extra as T),
			error: err.message
		});
	}
	throw err;
}

/** Reads `?page=N` and returns the backend limit and offset. */
export function pagination(url: URL, limit = 20) {
	const raw = Number(url.searchParams.get('page') ?? '1');
	const page = Number.isInteger(raw) && raw > 0 ? raw : 1;
	return { page, limit, offset: (page - 1) * limit };
}

export function pageCount(total: number, limit: number): number {
	return Math.max(1, Math.ceil(total / limit));
}

export function field(form: FormData, name: string): string {
	const value = form.get(name);
	return typeof value === 'string' ? value : '';
}
