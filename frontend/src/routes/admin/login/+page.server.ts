import { fail, redirect } from '@sveltejs/kit';
import { clientInfo, field } from '$lib/server/admin';
import { backendFetch, errorMessage } from '$lib/server/backend';
import { relaySetCookies } from '$lib/server/session';
import type { Actions } from './$types';

export const actions: Actions = {
	default: async (event) => {
		const form = await event.request.formData();
		const login = field(form, 'login').trim();
		const password = field(form, 'password');
		if (!login || !password) {
			return fail(400, { login, error: 'Enter login and password.' });
		}

		const client = clientInfo(event);
		const res = await backendFetch('/api/admin/login', {
			method: 'POST',
			headers: {
				'content-type': 'application/json',
				'user-agent': client.userAgent,
				'x-forwarded-for': client.clientAddress
			},
			body: JSON.stringify({ login, password })
		});

		if (res.status === 401) {
			await res.body?.cancel();
			return fail(401, { login, error: 'Invalid login or password.' });
		}
		if (res.status === 429) {
			await res.body?.cancel();
			return fail(429, { login, error: 'Too many login attempts. Please wait and try again.' });
		}
		if (!res.ok) {
			return fail(res.status, { login, error: await errorMessage(res) });
		}

		relaySetCookies(res.headers, event.cookies);
		await res.body?.cancel();
		redirect(303, '/admin');
	}
};
