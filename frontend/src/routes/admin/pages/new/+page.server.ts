import { fail, redirect } from '@sveltejs/kit';
import { actionFailed, adminApi } from '$lib/server/admin';
import { parseContent } from '$lib/server/content';
import type { AdminPage } from '$lib/types';
import type { Actions } from './$types';

export const actions: Actions = {
	save: async (event) => {
		const { values, error } = parseContent(await event.request.formData(), 'page');
		if (error) return fail(400, { error, values });
		let page: AdminPage;
		try {
			page = await adminApi(event).post<AdminPage>('/pages', values);
		} catch (err) {
			return actionFailed(err, { values });
		}
		redirect(303, `/admin/pages/${page.id}?created=1`);
	}
};
