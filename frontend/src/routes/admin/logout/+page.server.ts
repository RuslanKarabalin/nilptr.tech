import { redirect } from '@sveltejs/kit';
import { adminApi } from '$lib/server/admin';
import { ACCESS_COOKIE, REFRESH_COOKIE, relaySetCookies } from '$lib/server/session';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = () => redirect(303, '/admin');

export const actions: Actions = {
	default: async (event) => {
		try {
			const res = await adminApi(event).raw('/logout', { method: 'POST' });
			relaySetCookies(res.headers, event.cookies);
			await res.body?.cancel();
		} catch (err) {
			console.error('logout failed', err);
		}
		// Clear locally as well, even if the backend did not answer.
		event.cookies.delete(ACCESS_COOKIE, { path: '/' });
		event.cookies.delete(REFRESH_COOKIE, { path: '/admin' });
		redirect(303, '/admin/login');
	}
};
