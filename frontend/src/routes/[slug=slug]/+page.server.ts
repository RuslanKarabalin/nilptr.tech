import { error } from '@sveltejs/kit';
import { backendDown, getPage } from '$lib/server/api';
import { excerpt, renderMarkdown } from '$lib/server/markdown';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params }) => {
	const page = await getPage(params.slug).catch(backendDown);
	if (page) {
		return {
			page: { slug: page.slug, title: page.title, updated_at: page.updated_at },
			html: await renderMarkdown(page.body),
			description: excerpt(page.body)
		};
	}

	// /cv existed before pages were editable, keep the address alive until
	// a page with the slug `cv` is created in the admin.
	if (params.slug === 'cv') {
		return {
			page: { slug: 'cv', title: 'Curriculum Vitae', updated_at: null },
			html: '<p>Coming soon.</p>',
			description: 'Curriculum Vitae of Ruslan Karabalin.'
		};
	}

	error(404, 'Page not found');
};
