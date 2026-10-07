import {
	actionFailed,
	adminApi,
	field,
	loadFailed,
	pageCount,
	pagination
} from '$lib/server/admin';
import type { AdminPageSummary, List } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const { page, limit, offset } = pagination(event.url);
	try {
		const pages = await adminApi(event).get<List<AdminPageSummary>>('/pages', { limit, offset });
		return {
			pages: pages.items,
			total: pages.total,
			page,
			pageCount: pageCount(pages.total, limit)
		};
	} catch (err) {
		loadFailed(err);
	}
};

export const actions: Actions = {
	delete: async (event) => {
		const id = field(await event.request.formData(), 'id');
		try {
			await adminApi(event).del(`/pages/${encodeURIComponent(id)}`);
			return { success: 'Page deleted.' };
		} catch (err) {
			return actionFailed(err);
		}
	}
};
