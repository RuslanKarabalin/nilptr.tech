// Low level client for the Go backend, see docs/API.md.

import { env } from '$env/dynamic/private';

const DEFAULT_TIMEOUT_MS = 10_000;

export function backendUrl(): string {
	return (env.BACKEND_URL || 'http://localhost:8080').replace(/\/+$/, '');
}

export class ApiError extends Error {
	status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

/** Extracts the `{"error": "..."}` message from a failed response. */
export async function errorMessage(res: Response): Promise<string> {
	try {
		const text = await res.text();
		try {
			const body = JSON.parse(text);
			if (body && typeof body.error === 'string' && body.error) return body.error;
		} catch {
			// not JSON
		}
		if (text && text.length < 300) return text;
	} catch {
		// body could not be read
	}
	return `Request failed with status ${res.status}`;
}

export interface RequestOptions extends RequestInit {
	/** Timeout in milliseconds, 0 disables it. */
	timeout?: number;
	/** Query string parameters, undefined and empty values are skipped. */
	query?: Record<string, string | number | undefined | null>;
}

export function buildUrl(path: string, query?: RequestOptions['query']): string {
	const url = new URL(backendUrl() + path);
	for (const [key, value] of Object.entries(query ?? {})) {
		if (value !== undefined && value !== null && value !== '') {
			url.searchParams.set(key, String(value));
		}
	}
	return url.toString();
}

/** Sends a request to the backend and returns the raw response. */
export async function backendFetch(path: string, options: RequestOptions = {}): Promise<Response> {
	const { timeout = DEFAULT_TIMEOUT_MS, query, ...init } = options;
	const signal = init.signal ?? (timeout > 0 ? AbortSignal.timeout(timeout) : undefined);
	try {
		return await fetch(buildUrl(path, query), { ...init, signal });
	} catch (err) {
		throw new ApiError(503, `Backend is unavailable: ${(err as Error).message}`);
	}
}

/** Sends a request and parses JSON, throws ApiError on non 2xx statuses. */
export async function backendJson<T>(path: string, options: RequestOptions = {}): Promise<T> {
	const headers = new Headers(options.headers);
	headers.set('accept', 'application/json');
	if (options.body !== undefined && typeof options.body === 'string') {
		headers.set('content-type', 'application/json');
	}
	const res = await backendFetch(path, { ...options, headers });
	if (!res.ok) throw new ApiError(res.status, await errorMessage(res));
	if (res.status === 204) return undefined as T;
	const text = await res.text();
	return (text ? JSON.parse(text) : undefined) as T;
}
