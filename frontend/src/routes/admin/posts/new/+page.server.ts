import { fail, redirect } from '@sveltejs/kit';
import { actionFailed, adminApi } from '$lib/server/admin';
import { parseContent } from '$lib/server/content';
import type { AdminPost } from '$lib/types';
import type { Actions } from './$types';

export const actions: Actions = {
	save: async (event) => {
		const { values, error } = parseContent(await event.request.formData(), 'post');
		if (error) return fail(400, { error, values });
		let post: AdminPost;
		try {
			post = await adminApi(event).post<AdminPost>('/posts', values);
		} catch (err) {
			return actionFailed(err, { values });
		}
		redirect(303, `/admin/posts/${post.id}?created=1`);
	}
};
