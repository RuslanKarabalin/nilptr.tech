import { adminApi, loadFailed } from '$lib/server/admin';
import type { AdminComment, AdminFile, AdminPageSummary, AdminPostSummary, List } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const api = adminApi(event);
	try {
		const [posts, drafts, pages, comments, files] = await Promise.all([
			api.get<List<AdminPostSummary>>('/posts', { limit: 5 }),
			api.get<List<AdminPostSummary>>('/posts', { limit: 1, status: 'draft' }),
			api.get<List<AdminPageSummary>>('/pages', { limit: 1 }),
			api.get<List<AdminComment>>('/comments', { limit: 1, status: 'pending' }),
			api.get<List<AdminFile>>('/files', { limit: 1 })
		]);
		return {
			recent: posts.items,
			counts: {
				posts: posts.total,
				drafts: drafts.total,
				pages: pages.total,
				pending: comments.total,
				files: files.total
			}
		};
	} catch (err) {
		loadFailed(err);
	}
};
