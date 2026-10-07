import { fail } from '@sveltejs/kit';
import {
	actionFailed,
	adminApi,
	field,
	loadFailed,
	pageCount,
	pagination
} from '$lib/server/admin';
import type { AdminComment, CommentStatus, List } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

const STATUSES: CommentStatus[] = ['pending', 'approved', 'rejected'];

export const load: PageServerLoad = async (event) => {
	const { page, limit, offset } = pagination(event.url);
	const raw = event.url.searchParams.get('status') as CommentStatus | null;
	const status = raw && STATUSES.includes(raw) ? raw : 'pending';
	try {
		const comments = await adminApi(event).get<List<AdminComment>>('/comments', {
			status,
			limit,
			offset
		});
		return {
			comments: comments.items,
			total: comments.total,
			status,
			statuses: STATUSES,
			page,
			pages: pageCount(comments.total, limit)
		};
	} catch (err) {
		loadFailed(err);
	}
};

async function moderate(
	event: Parameters<Actions[string]>[0],
	action: 'approve' | 'reject' | 'delete'
) {
	const id = Number(field(await event.request.formData(), 'id'));
	if (!Number.isInteger(id)) return fail(400, { error: 'Invalid comment id.' });
	const api = adminApi(event);
	try {
		if (action === 'delete') await api.del(`/comments/${id}`);
		else await api.post(`/comments/${id}/${action}`);
		const done = { approve: 'approved', reject: 'rejected', delete: 'deleted' }[action];
		return { success: `Comment ${done}.` };
	} catch (err) {
		return actionFailed(err);
	}
}

export const actions: Actions = {
	approve: (event) => moderate(event, 'approve'),
	reject: (event) => moderate(event, 'reject'),
	delete: (event) => moderate(event, 'delete')
};
