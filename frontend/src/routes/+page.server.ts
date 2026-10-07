import { error } from '@sveltejs/kit';
import { backendDown, listPosts } from '$lib/server/api';
import type { PageServerLoad } from './$types';

const PAGE_SIZE = 10;

export const load: PageServerLoad = async ({ url }) => {
	const raw = url.searchParams.get('page');
	const pageNumber = raw === null ? 1 : Number(raw);
	if (!Number.isInteger(pageNumber) || pageNumber < 1) error(404, 'Page not found');

	const posts = await listPosts(PAGE_SIZE, (pageNumber - 1) * PAGE_SIZE).catch(backendDown);
	const pages = Math.max(1, Math.ceil(posts.total / PAGE_SIZE));
	if (pageNumber > pages) error(404, 'Page not found');

	return { posts: posts.items, page: pageNumber, pages };
};
