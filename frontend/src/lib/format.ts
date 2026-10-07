// Formatting helpers shared by the server and the browser.
// They avoid locale dependent output so SSR and hydration always match.

export function formatDate(value: string | null | undefined): string {
	if (!value) return '';
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return '';
	return date.toISOString().slice(0, 10);
}

export function formatDateTime(value: string | null | undefined): string {
	if (!value) return '';
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return '';
	return date.toISOString().slice(0, 16).replace('T', ' ') + ' UTC';
}

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB'];

export function formatSize(bytes: number): string {
	if (!Number.isFinite(bytes) || bytes < 0) return '';
	let value = bytes;
	let unit = 0;
	while (value >= 1024 && unit < UNITS.length - 1) {
		value /= 1024;
		unit++;
	}
	const digits = unit === 0 || value >= 10 ? 0 : 1;
	return `${value.toFixed(digits)} ${UNITS[unit]}`;
}

/** Markdown directive that embeds a stored file into a post. */
export function directiveFor(file: { id: string; content_type: string }): string {
	if (file.content_type.startsWith('image/')) return `::image{id=${file.id} alt=""}`;
	if (file.content_type.startsWith('video/')) return `::video{id=${file.id}}`;
	return `::file{id=${file.id}}`;
}

/** True for links that point to this site (they start with a single slash). */
export function isInternalUrl(url: string): boolean {
	return url.startsWith('/') && !url.startsWith('//');
}
