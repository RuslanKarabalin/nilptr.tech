import { fail, redirect } from '@sveltejs/kit';
import { actionFailed, adminApi, loadFailed } from '$lib/server/admin';
import { parseContent } from '$lib/server/content';
import type { AdminPage } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	try {
		const page = await adminApi(event).get<AdminPage>(
			`/pages/${encodeURIComponent(event.params.id)}`
		);
		return { page, created: event.url.searchParams.has('created') };
	} catch (err) {
		loadFailed(err);
	}
};

export const actions: Actions = {
	save: async (event) => {
		const { values, error } = parseContent(await event.request.formData(), 'page');
		if (error) return fail(400, { error, values });
		try {
			await adminApi(event).put<AdminPage>(`/pages/${encodeURIComponent(event.params.id)}`, values);
			return { success: 'Saved.' };
		} catch (err) {
			return actionFailed(err, { values });
		}
	},
	delete: async (event) => {
		try {
			await adminApi(event).del(`/pages/${encodeURIComponent(event.params.id)}`);
		} catch (err) {
			return actionFailed(err);
		}
		redirect(303, '/admin/pages');
	}
};
