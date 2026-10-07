// Markdown to HTML on the server, see the Markdown section of docs/API.md.
// Used for public posts and pages and for the admin preview.

import type { Element, ElementContent } from 'hast';
import type { Nodes, Paragraph, Parent, PhrasingContent, Root, RootContent } from 'mdast';
import type { ContainerDirective, LeafDirective, TextDirective } from 'mdast-util-directive';
import rehypeKatex from 'rehype-katex';
import rehypeSanitize, { defaultSchema, type Options as SanitizeSchema } from 'rehype-sanitize';
import rehypeStringify from 'rehype-stringify';
import remarkDirective from 'remark-directive';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import { unified } from 'unified';
import { SKIP, visit } from 'unist-util-visit';
import { formatSize } from '$lib/format';
import type { FileMeta } from '$lib/types';
import { getFileMeta } from './api';

export type FileLoader = (id: string) => Promise<FileMeta | null>;

export interface MarkdownOptions {
	/** Loads metadata for `::file`, defaults to GET /api/files/{id}. */
	loadFile?: FileLoader;
}

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

type Directive = ContainerDirective | LeafDirective | TextDirective;

function isDirective(node: Nodes): node is Directive {
	return (
		node.type === 'containerDirective' ||
		node.type === 'leafDirective' ||
		node.type === 'textDirective'
	);
}

function text(value: string): ElementContent {
	return { type: 'text', value };
}

function el(
	tagName: string,
	properties: Element['properties'],
	children: ElementContent[] = []
): Element {
	return { type: 'element', tagName, properties, children };
}

function setHast(node: Directive, element: Element) {
	node.data = {
		...node.data,
		hName: element.tagName,
		hProperties: element.properties,
		hChildren: element.children
	};
}

function fileCard(id: string, meta: FileMeta | null): Element {
	if (!meta) {
		return el('div', { className: ['file-card', 'file-card-missing'] }, [
			text(`File ${id} not found`)
		]);
	}
	return el('div', { className: ['file-card'] }, [
		el('a', { href: meta.url, className: ['file-card-name'] }, [text(meta.name)]),
		text(' '),
		el('span', { className: ['file-card-size'] }, [text(formatSize(meta.size))])
	]);
}

/** Turns an unknown directive back into its source text, for example `10:30` or `a:b`. */
function restoreDirective(prefix: string, node: Directive): PhrasingContent[] {
	const restored: PhrasingContent[] = [{ type: 'text', value: `${prefix}${node.name}` }];
	if (node.children.length) {
		restored.push({ type: 'text', value: '[' }, ...(node.children as PhrasingContent[]), {
			type: 'text',
			value: ']'
		});
	}
	return restored;
}

/** Remark plugin for `::image`, `::video` and `::file`. */
function remarkEmbeds(loadFile: FileLoader) {
	return async (tree: Root) => {
		const files: { node: LeafDirective; id: string }[] = [];

		visit(tree, (node, index, parent) => {
			if (!isDirective(node as Nodes)) return;
			const directive = node as Directive;
			const container = parent as Parent | undefined;

			if (directive.type === 'textDirective') {
				if (container && index !== undefined) {
					const restored = restoreDirective(':', directive);
					container.children.splice(index, 1, ...(restored as RootContent[]));
					return [SKIP, index + restored.length];
				}
				return;
			}

			if (directive.type === 'containerDirective') {
				// Unknown container: keep the content in a plain div.
				directive.data = { ...directive.data, hName: 'div', hProperties: {} };
				return;
			}

			const id = String(directive.attributes?.id ?? '');
			const valid = UUID.test(id);

			if (directive.name === 'image' && valid) {
				const alt = String(directive.attributes?.alt ?? '');
				setHast(directive, el('img', { src: `/files/${id}`, alt, loading: 'lazy' }));
			} else if (directive.name === 'video' && valid) {
				setHast(
					directive,
					el('video', { controls: true, preload: 'metadata', src: `/files/${id}` })
				);
			} else if (directive.name === 'file' && valid) {
				files.push({ node: directive, id: id.toLowerCase() });
			} else if (['image', 'video', 'file'].includes(directive.name)) {
				setHast(directive, el('p', {}, [text(`[::${directive.name}: invalid or missing id]`)]));
			} else if (container && index !== undefined) {
				const paragraph: Paragraph = {
					type: 'paragraph',
					children: restoreDirective('::', directive)
				};
				container.children.splice(index, 1, paragraph as RootContent);
				return SKIP;
			}
		});

		const unique = [...new Set(files.map((f) => f.id))];
		const metas = new Map(
			await Promise.all(
				unique.map(async (id) => {
					try {
						return [id, await loadFile(id)] as const;
					} catch {
						return [id, null] as const;
					}
				})
			)
		);
		for (const { node, id } of files) {
			setHast(node, fileCard(id, metas.get(id) ?? null));
		}
	};
}

const MATHML_TAGS = [
	'math',
	'semantics',
	'annotation',
	'mrow',
	'mi',
	'mn',
	'mo',
	'ms',
	'mtext',
	'mspace',
	'msup',
	'msub',
	'msubsup',
	'mfrac',
	'msqrt',
	'mroot',
	'mover',
	'munder',
	'munderover',
	'mtable',
	'mtr',
	'mtd',
	'mlabeledtr',
	'mstyle',
	'mpadded',
	'mphantom',
	'menclose',
	'merror'
];

const MATHML_ATTRIBUTES = [
	'xmlns',
	'display',
	'encoding',
	'mathvariant',
	'mathcolor',
	'mathbackground',
	'mathsize',
	'stretchy',
	'fence',
	'separator',
	'lspace',
	'rspace',
	'accent',
	'accentunder',
	'largeop',
	'movablelimits',
	'symmetric',
	'minsize',
	'maxsize',
	'linethickness',
	'scriptlevel',
	'displaystyle',
	'columnalign',
	'columnspacing',
	'columnlines',
	'rowalign',
	'rowspacing',
	'rowlines',
	'frame',
	'framespacing',
	'equalrows',
	'equalcolumns',
	'side',
	'notation',
	'width',
	'height',
	'depth',
	'voffset'
];

/**
 * GitHub style default schema plus KaTeX output (HTML spans with classes
 * and inline styles, MathML, the SVG used for radicals and stretchy arrows)
 * and the elements produced by the directives above.
 */
export const sanitizeSchema: SanitizeSchema = {
	...defaultSchema,
	tagNames: [...(defaultSchema.tagNames ?? []), 'video', 'svg', 'path', 'line', ...MATHML_TAGS],
	attributes: {
		...defaultSchema.attributes,
		a: [
			...(defaultSchema.attributes?.a ?? []).filter(
				(a) => !Array.isArray(a) || a[0] !== 'className'
			),
			['className', 'data-footnote-backref', 'file-card-name']
		],
		div: [
			...(defaultSchema.attributes?.div ?? []),
			['className', 'file-card', 'file-card-missing', 'math', 'math-display']
		],
		img: [...(defaultSchema.attributes?.img ?? []), 'alt', ['loading', 'lazy']],
		video: ['src', 'controls', ['preload', 'metadata']],
		span: ['className', 'style', 'ariaHidden'],
		code: [['className', /^language-./, 'math-inline', 'math-display']],
		svg: [
			'xmlns',
			'width',
			'height',
			'viewBox',
			'viewbox',
			'preserveAspectRatio',
			'preserveaspectratio',
			'style'
		],
		path: ['d'],
		line: ['x1', 'y1', 'x2', 'y2', 'strokeWidth', 'stroke-width'],
		...Object.fromEntries(MATHML_TAGS.map((tag) => [tag, MATHML_ATTRIBUTES]))
	}
};

export async function renderMarkdown(
	source: string,
	options: MarkdownOptions = {}
): Promise<string> {
	const file = await unified()
		.use(remarkParse)
		.use(remarkGfm)
		.use(remarkMath)
		.use(remarkDirective)
		.use(remarkEmbeds, options.loadFile ?? getFileMeta)
		.use(remarkRehype)
		.use(rehypeKatex, { throwOnError: false, strict: false })
		.use(rehypeSanitize, sanitizeSchema)
		.use(rehypeStringify)
		.process(source);
	return String(file);
}

/** Plain text excerpt of markdown for `<meta name="description">`. */
export function excerpt(source: string, max = 160): string {
	const plain = source
		.replace(/```[\s\S]*?```/g, ' ')
		.replace(/\$\$[\s\S]*?\$\$/g, ' ')
		.split('\n')
		.filter((line) => !/^\s*(#|::|\||>|-{3,}|\*{3,})/.test(line))
		.join(' ')
		.replace(/!\[[^\]]*\]\([^)]*\)/g, ' ')
		.replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
		.replace(/\$[^$]*\$/g, ' ')
		.replace(/[*_`~]/g, '')
		.replace(/\s+/g, ' ')
		.trim();
	if (plain.length <= max) return plain;
	const cut = plain.slice(0, max - 3);
	const space = cut.lastIndexOf(' ');
	return (space > max / 2 ? cut.slice(0, space) : cut) + '...';
}
