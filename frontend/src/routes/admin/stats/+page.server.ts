import { adminApi, loadFailed } from '$lib/server/admin';
import type { List, StatsItem } from '$lib/types';
import type { PageServerLoad } from './$types';

const DATE = /^\d{4}-\d{2}-\d{2}$/;
const DAY_MS = 24 * 60 * 60 * 1000;

function isoDate(date: Date): string {
	return date.toISOString().slice(0, 10);
}

function validDate(value: string | null): value is string {
	return !!value && DATE.test(value) && !Number.isNaN(new Date(value).getTime());
}

export const load: PageServerLoad = async (event) => {
	const today = new Date();
	const qFrom = event.url.searchParams.get('from');
	const qTo = event.url.searchParams.get('to');
	let to = validDate(qTo) ? qTo : isoDate(today);
	let from = validDate(qFrom) ? qFrom : isoDate(new Date(new Date(to).getTime() - 29 * DAY_MS));
	if (from > to) [from, to] = [to, from];

	try {
		const stats = await adminApi(event).get<List<StatsItem>>('/stats', { from, to });
		return { from, to, items: stats.items, total: stats.total };
	} catch (err) {
		loadFailed(err);
	}
};
