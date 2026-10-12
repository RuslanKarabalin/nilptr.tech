import type { Nav } from '$lib/types';

/** Links shown when GET /api/nav fails. */
export const FALLBACK_NAV: Nav = {
	header: [
		{ label: 'CV', url: '/cv' },
		{ label: 'GitHub', url: 'https://github.com/RuslanKarabalin' }
	],
	footer: [{ label: 'GitHub', url: 'https://github.com/RuslanKarabalin' }]
};

export const SITE_NAME = 'nilptr.tech';
export const SITE_DESCRIPTION = 'Personal blog of Ruslan Karabalin.';

/** Headline on the home page; the part after the | is shown in the accent color. */
export const SITE_TAGLINE = 'Notes on Go, infrastructure | and the things in between';
