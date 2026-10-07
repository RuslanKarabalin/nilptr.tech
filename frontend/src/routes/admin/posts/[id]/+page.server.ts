import { fail, redirect } from '@sveltejs/kit';
import { actionFailed, adminApi, loadFailed } from '$lib/server/admin';
import { parseContent } from '$lib/server/content';
import type { AdminPost } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	try {
		const post = await adminApi(event).get<AdminPost>(
			`/posts/${encodeURIComponent(event.params.id)}`
		);
		return { post, created: event.url.searchParams.has('created') };
	} catch (err) {
		loadFailed(err);
	}
};

export const actions: Actions = {
	save: async (event) => {
		const { values, error } = parseContent(await event.request.formData(), 'post');
		if (error) return fail(400, { error, values });
		try {
			await adminApi(event).put<AdminPost>(`/posts/${encodeURIComponent(event.params.id)}`, values);
			return { success: 'Saved.' };
		} catch (err) {
			return actionFailed(err, { values });
		}
	},
	delete: async (event) => {
		try {
			await adminApi(event).del(`/posts/${encodeURIComponent(event.params.id)}`);
		} catch (err) {
			return actionFailed(err);
		}
		redirect(303, '/admin/posts');
	}
};
