import { error, fail } from '@sveltejs/kit';
import { commentPayload, commentResult } from '$lib/comments';
import { clientInfo } from '$lib/server/admin';
import { backendDown, getPost, listComments } from '$lib/server/api';
import { backendFetch, errorMessage } from '$lib/server/backend';
import { excerpt, renderMarkdown } from '$lib/server/markdown';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params }) => {
	const post = await getPost(params.slug).catch(backendDown);
	if (!post) error(404, 'Post not found');

	const [html, comments] = await Promise.all([
		renderMarkdown(post.body),
		listComments(post.slug).catch((err) => {
			console.error('GET comments failed', err);
			return null;
		})
	]);

	return {
		post: {
			slug: post.slug,
			title: post.title,
			status: post.status,
			published_at: post.published_at,
			updated_at: post.updated_at
		},
		html,
		description: excerpt(post.body),
		comments
	};
};

export const actions: Actions = {
	// Fallback for browsers without JavaScript. With JavaScript the form
	// posts straight to the backend from the browser.
	comment: async (event) => {
		const form = await event.request.formData();
		const payload = commentPayload(
			String(form.get('author') ?? ''),
			String(form.get('body') ?? ''),
			String(form.get('website') ?? '')
		);
		const client = clientInfo(event);
		const res = await backendFetch(`/api/posts/${encodeURIComponent(event.params.slug)}/comments`, {
			method: 'POST',
			headers: {
				'content-type': 'application/json',
				'user-agent': client.userAgent,
				'x-forwarded-for': client.clientAddress
			},
			body: JSON.stringify(payload)
		});
		const result = commentResult(res.status, res.ok ? undefined : await errorMessage(res));
		return result.ok ? { comment: result } : fail(res.status, { comment: result });
	}
};
