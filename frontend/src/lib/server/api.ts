// Public backend endpoints used by server load functions.

import { error } from '@sveltejs/kit';
import type { Comment, FileMeta, List, Nav, Page, Post, PostSummary } from '$lib/types';
import { ApiError, backendJson } from './backend';

export async function listPosts(limit: number, offset: number): Promise<List<PostSummary>> {
	return backendJson('/api/posts', { query: { limit, offset } });
}

/** Returns null when the backend answers 404. */
async function orNull<T>(promise: Promise<T>): Promise<T | null> {
	try {
		return await promise;
	} catch (err) {
		if (err instanceof ApiError && err.status === 404) return null;
		throw err;
	}
}

export function getPost(slug: string): Promise<Post | null> {
	return orNull(backendJson(`/api/posts/${encodeURIComponent(slug)}`));
}

export function listComments(slug: string): Promise<Comment[]> {
	return backendJson(`/api/posts/${encodeURIComponent(slug)}/comments`);
}

export function getPage(slug: string): Promise<Page | null> {
	return orNull(backendJson(`/api/pages/${encodeURIComponent(slug)}`));
}

export function getNav(): Promise<Nav> {
	return backendJson('/api/nav', { timeout: 3000 });
}

export function getFileMeta(id: string): Promise<FileMeta | null> {
	return orNull(backendJson(`/api/files/${encodeURIComponent(id)}`, { timeout: 5000 }));
}

/** Turns a failed backend call in a public load into a 503 error page. */
export function backendDown(err: unknown): never {
	console.error('backend request failed', err);
	error(503, 'The site is temporarily unavailable. Please try again later.');
}
