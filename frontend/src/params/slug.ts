import type { ParamMatcher } from '@sveltejs/kit';

export const SLUG = /^[a-z0-9]+(-[a-z0-9]+)*$/;

export const match: ParamMatcher = (param) => SLUG.test(param);
