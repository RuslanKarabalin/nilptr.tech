import {
	actionFailed,
	adminApi,
	field,
	loadFailed,
	pageCount,
	pagination
} from '$lib/server/admin';
import { POST_STATUSES, type AdminPostSummary, type List, type PostStatus } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const { page, limit, offset } = pagination(event.url);
	const raw = event.url.searchParams.get('status') as PostStatus | null;
	const status = raw && POST_STATUSES.includes(raw) ? raw : '';
	try {
		const posts = await adminApi(event).get<List<AdminPostSummary>>('/posts', {
			limit,
			offset,
			status
		});
		return {
			posts: posts.items,
			total: posts.total,
			page,
			pages: pageCount(posts.total, limit),
			status
		};
	} catch (err) {
		loadFailed(err);
	}
};

export const actions: Actions = {
	delete: async (event) => {
		const id = field(await event.request.formData(), 'id');
		try {
			await adminApi(event).del(`/posts/${encodeURIComponent(id)}`);
			return { success: 'Post deleted.' };
		} catch (err) {
			return actionFailed(err);
		}
	}
};
