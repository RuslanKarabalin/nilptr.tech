import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('$env/dynamic/private', () => ({ env: { BACKEND_URL: 'http://backend.test/' } }));

import { excerpt, renderMarkdown } from './markdown';

const ID = '0b4e5d1c-2f4a-4c3b-9a1e-123456789abc';

const noFiles = { loadFile: async () => null };

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('renderMarkdown', () => {
	it('renders GFM', async () => {
		const html = await renderMarkdown('| a | b |\n| - | - |\n| 1 | 2 |\n\n~~old~~', noFiles);
		expect(html).toContain('<table>');
		expect(html).toContain('<del>old</del>');
	});

	it('renders inline and display math with KaTeX', async () => {
		const html = await renderMarkdown('Euler $e^{i\\pi} + 1 = 0$\n\n$$\n\\sqrt{2}\n$$\n', noFiles);
		expect(html).toContain('class="katex"');
		expect(html).toContain('class="katex-display"');
		expect(html).toContain('<math xmlns="http://www.w3.org/1998/Math/MathML"');
		expect(html).toContain(
			'<annotation encoding="application/x-tex">e^{i\\pi} + 1 = 0</annotation>'
		);
		// inline styles and the radical SVG survive the sanitizer
		expect(html).toMatch(/<span class="strut" style="height:[^"]+">/);
		expect(html).toContain('<svg xmlns="http://www.w3.org/2000/svg"');
		expect(html).toContain('<path d="');
	});

	it('does not fail on invalid math', async () => {
		const html = await renderMarkdown('$\\frac{$', noFiles);
		expect(html).toContain('katex-error');
	});

	it('strips script tags and dangerous attributes', async () => {
		const html = await renderMarkdown(
			'<script>alert(1)</script>\n\n<img src=x onerror="alert(2)">\n\n[x](javascript:alert(3))\n\n<a href="/" onclick="alert(4)">a</a>',
			noFiles
		);
		expect(html).not.toContain('<script');
		expect(html).not.toContain('alert(1)');
		expect(html).not.toContain('onerror');
		expect(html).not.toContain('onclick');
		expect(html).not.toContain('javascript:');
	});

	it('renders ::image', async () => {
		const html = await renderMarkdown(`::image{id=${ID} alt="A cat"}`, noFiles);
		expect(html).toBe(`<img src="/files/${ID}" alt="A cat" loading="lazy">`);
	});

	it('renders ::video', async () => {
		const html = await renderMarkdown(`::video{id=${ID}}`, noFiles);
		expect(html).toBe(`<video controls preload="metadata" src="/files/${ID}"></video>`);
	});

	it('rejects directives with an invalid id', async () => {
		const html = await renderMarkdown('::image{id="x onerror=alert(1)"}', noFiles);
		expect(html).toBe('<p>[::image: invalid or missing id]</p>');
	});

	it('keeps unknown directives as text', async () => {
		const html = await renderMarkdown('At 10:30 see a:b[c]\n\n::foo[bar]', noFiles);
		expect(html).toBe('<p>At 10:30 see a:b[c]</p>\n<p>::foo[bar]</p>');
	});

	it('renders ::file with metadata fetched from the backend', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = String(input);
			if (url === `http://backend.test/api/files/${ID}`) {
				return Response.json({
					id: ID,
					name: 'report <final>.pdf',
					content_type: 'application/pdf',
					size: 1536,
					url: `/files/${ID}/report%20%3Cfinal%3E.pdf`
				});
			}
			return Response.json({ error: 'not found' }, { status: 404 });
		});
		vi.stubGlobal('fetch', fetchMock);

		const html = await renderMarkdown(`::file{id=${ID}}\n\n::file{id=${ID}}`);
		const card =
			`<div class="file-card"><a href="/files/${ID}/report%20%3Cfinal%3E.pdf" class="file-card-name">` +
			'report &#x3C;final>.pdf</a> <span class="file-card-size">1.5 KB</span></div>';
		expect(html).toBe(`${card}\n${card}`);
		// the same id is fetched only once
		expect(fetchMock).toHaveBeenCalledTimes(1);
	});

	it('renders a placeholder when the file is missing', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => Response.json({ error: 'not found' }, { status: 404 }))
		);
		const html = await renderMarkdown(`::file{id=${ID}}`);
		expect(html).toBe(`<div class="file-card file-card-missing">File ${ID} not found</div>`);
	});

	it('renders a placeholder when the backend is down', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => {
				throw new TypeError('fetch failed');
			})
		);
		const html = await renderMarkdown(`::file{id=${ID}}`);
		expect(html).toContain('file-card-missing');
	});
});

describe('excerpt', () => {
	it('builds a plain text description', () => {
		const text = excerpt('# Title\n\nSome **bold** [link](/x) and $x^2$ math.\n\n::image{id=1}');
		expect(text).toBe('Some bold link and math.');
	});

	it('truncates long text', () => {
		const text = excerpt('word '.repeat(100), 20);
		expect(text.length).toBeLessThanOrEqual(20);
		expect(text.endsWith('...')).toBe(true);
	});
});
