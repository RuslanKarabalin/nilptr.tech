import type { RequestHandler } from './$types';

// The frontend has no own dependencies to check: it renders an error page
// itself when the backend is down, so it is ready as soon as it serves.
export const GET: RequestHandler = () =>
	new Response('ok', { headers: { 'cache-control': 'no-store' } });
