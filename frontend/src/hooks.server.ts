import { error, json, redirect, type Handle } from '@sveltejs/kit';
import { clientInfo } from '$lib/server/admin';
import { ApiError, backendUrl } from '$lib/server/backend';
import { resolveAdmin } from '$lib/server/session';

const LOGIN_PATH = '/admin/login';

function isAdminPath(pathname: string): boolean {
	return pathname === '/admin' || pathname.startsWith('/admin/');
}

export const handle: Handle = async ({ event, resolve }) => {
	event.locals.user = null;
	const { pathname } = event.url;

	if (!isAdminPath(pathname)) return resolve(event);

	try {
		// Refreshes the tokens at most once and relays Set-Cookie to the
		// browser before any load function or action runs.
		event.locals.user = await resolveAdmin({
			cookies: event.cookies,
			client: clientInfo(event),
			fetch,
			baseUrl: backendUrl()
		});
	} catch (err) {
		console.error('admin session check failed', err);
		error(503, err instanceof ApiError ? err.message : 'Backend is unavailable');
	}

	if (pathname === LOGIN_PATH || pathname.startsWith(LOGIN_PATH + '/')) {
		if (event.locals.user && event.request.method === 'GET') redirect(303, '/admin');
	} else if (!event.locals.user) {
		if (pathname === '/admin/preview') {
			return json({ error: 'unauthorized' }, { status: 401 });
		}
		if (event.request.headers.get('x-sveltekit-action') === 'true') {
			// A use:enhance form submission expects an ActionResult, not a 303.
			return json({ type: 'redirect', status: 303, location: LOGIN_PATH });
		}
		redirect(303, LOGIN_PATH);
	}

	return resolve(event);
};
