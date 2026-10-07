import { fail, redirect } from '@sveltejs/kit';
import { adminApi, field, loadFailed } from '$lib/server/admin';
import { errorMessage } from '$lib/server/backend';
import { ACCESS_COOKIE, REFRESH_COOKIE, relaySetCookies } from '$lib/server/session';
import type { AdminSession } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	try {
		const sessions = await adminApi(event).get<AdminSession[]>('/sessions');
		return { sessions };
	} catch (err) {
		loadFailed(err);
	}
};

export const actions: Actions = {
	end: async (event) => {
		const form = await event.request.formData();
		const id = field(form, 'id');
		const current = field(form, 'current') === 'true';
		const res = await adminApi(event).raw(`/sessions/${encodeURIComponent(id)}`, {
			method: 'DELETE'
		});
		// Ending the current session clears both cookies on the backend side.
		const relayed = relaySetCookies(res.headers, event.cookies);
		if (!res.ok) {
			if (res.status === 401) redirect(303, '/admin/login');
			return fail(res.status, { error: await errorMessage(res) });
		}
		await res.body?.cancel();
		if (current) {
			if (relayed.length === 0) {
				event.cookies.delete(ACCESS_COOKIE, { path: '/' });
				event.cookies.delete(REFRESH_COOKIE, { path: '/admin' });
			}
			redirect(303, '/admin/login');
		}
		return { success: 'Session ended.' };
	}
};
