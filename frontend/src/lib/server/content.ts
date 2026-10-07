// Form parsing shared by the admin post and page editors.

import { SLUG } from '../../params/slug';
import { POST_STATUSES, type PostStatus } from '$lib/types';
import { field } from './admin';

export interface ContentInput {
	slug: string;
	title: string;
	body: string;
	status?: PostStatus;
}

/** Top level paths that are not pages, a page with such a slug would be unreachable. */
const RESERVED_PAGE_SLUGS = ['admin', 'api', 'files', 'posts', 'healthz', 'readyz'];

export function parseContent(
	form: FormData,
	kind: 'post' | 'page'
): { values: ContentInput; error: string | null } {
	const values: ContentInput = {
		slug: field(form, 'slug').trim(),
		title: field(form, 'title').trim(),
		body: field(form, 'body')
	};
	if (kind === 'post') {
		const status = field(form, 'status') as PostStatus;
		values.status = POST_STATUSES.includes(status) ? status : 'draft';
	}
	let error: string | null = null;
	if (!values.title) error = 'Title is required.';
	else if (!SLUG.test(values.slug)) {
		error = 'Slug must contain lowercase letters, digits and single dashes only.';
	} else if (kind === 'page' && RESERVED_PAGE_SLUGS.includes(values.slug)) {
		error = `Slug "${values.slug}" is reserved by the site.`;
	}
	return { values, error };
}
