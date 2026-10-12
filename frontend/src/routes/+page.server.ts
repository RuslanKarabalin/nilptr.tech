import { error } from '@sveltejs/kit';
import { readingMinutes } from '$lib/format';
import { backendDown, listPosts } from '$lib/server/api';
import { excerpt } from '$lib/server/markdown';
import type { PageServerLoad } from './$types';

const PAGE_SIZE = 10;

/** Drops an unclosed code fence or math block left by cutting the lead. */
function closedLead(lead: string): string {
	let text = lead;
	for (const fence of ['```', '$$']) {
		const count = text.split(fence).length - 1;
		if (count % 2 === 1) text = text.slice(0, text.lastIndexOf(fence));
	}
	return text;
}

export const load: PageServerLoad = async ({ url }) => {
	const raw = url.searchParams.get('page');
	const pageNumber = raw === null ? 1 : Number(raw);
	if (!Number.isInteger(pageNumber) || pageNumber < 1) error(404, 'Page not found');

	const posts = await listPosts(PAGE_SIZE, (pageNumber - 1) * PAGE_SIZE).catch(backendDown);
	const pages = Math.max(1, Math.ceil(posts.total / PAGE_SIZE));
	if (pageNumber > pages) error(404, 'Page not found');

	return {
		posts: posts.items.map((post) => ({
			slug: post.slug,
			title: post.title,
			published_at: post.published_at,
			excerpt: excerpt(closedLead(post.lead ?? ''), 220),
			minutes: readingMinutes(post.words ?? 0)
		})),
		page: pageNumber,
		pages,
		total: posts.total
	};
};
