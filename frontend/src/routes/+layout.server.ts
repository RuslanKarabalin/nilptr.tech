import { getNav } from '$lib/server/api';
import { FALLBACK_NAV } from '$lib/nav';
import type { Nav } from '$lib/types';
import type { LayoutServerLoad } from './$types';

function isNav(value: unknown): value is Nav {
	const nav = value as Nav | null;
	return !!nav && Array.isArray(nav.header) && Array.isArray(nav.footer);
}

export const load: LayoutServerLoad = async () => {
	try {
		const nav = await getNav();
		if (isNav(nav)) return { nav };
	} catch (err) {
		console.error('GET /api/nav failed', err);
	}
	return { nav: FALLBACK_NAV };
};
