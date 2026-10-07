import { fail } from '@sveltejs/kit';
import { actionFailed, adminApi, loadFailed } from '$lib/server/admin';
import type { AdminNav, NavLink } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

const LOCATIONS = ['header', 'footer'] as const;

export const load: PageServerLoad = async (event) => {
	try {
		return { nav: await adminApi(event).get<AdminNav>('/nav') };
	} catch (err) {
		loadFailed(err);
	}
};

function validUrl(url: string): boolean {
	return (
		(url.startsWith('/') && !url.startsWith('//')) ||
		/^https?:\/\/[^\s]+$/i.test(url) ||
		/^mailto:[^\s]+$/i.test(url)
	);
}

export const actions: Actions = {
	save: async (event) => {
		const form = await event.request.formData();
		const nav = { header: [] as NavLink[], footer: [] as NavLink[] };
		for (const location of LOCATIONS) {
			const labels = form.getAll(`${location}_label`).map((v) => String(v).trim());
			const urls = form.getAll(`${location}_url`).map((v) => String(v).trim());
			for (let i = 0; i < Math.max(labels.length, urls.length); i++) {
				const label = labels[i] ?? '';
				const url = urls[i] ?? '';
				if (!label && !url) continue;
				nav[location].push({ label, url });
			}
		}

		for (const location of LOCATIONS) {
			for (const item of nav[location]) {
				if (!item.label || !item.url) {
					return fail(400, { error: `Every ${location} item needs a label and a URL.`, nav });
				}
				if (!validUrl(item.url)) {
					return fail(400, {
						error: `"${item.url}" is not valid: use /path, http(s):// or mailto: links.`,
						nav
					});
				}
			}
		}

		try {
			await adminApi(event).put('/nav', nav);
			return { success: 'Navigation saved.' };
		} catch (err) {
			return actionFailed(err, { nav });
		}
	}
};
