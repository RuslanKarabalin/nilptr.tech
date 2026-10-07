import { error, json } from '@sveltejs/kit';
import { renderMarkdown } from '$lib/server/markdown';
import type { RequestHandler } from './$types';

const MAX_BODY = 1_000_000;

export const POST: RequestHandler = async ({ request, locals }) => {
	if (!locals.user) error(401, 'Unauthorized');
	let body: unknown;
	try {
		body = (await request.json())?.body;
	} catch {
		error(400, 'Invalid JSON');
	}
	if (typeof body !== 'string') error(400, 'Field body must be a string');
	if (body.length > MAX_BODY) error(413, 'Body is too large');
	return json({ html: await renderMarkdown(body) });
};
