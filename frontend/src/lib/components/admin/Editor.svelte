<script lang="ts">
	import { enhance } from '$app/forms';
	import Markdown from '$lib/components/Markdown.svelte';
	import { POST_STATUSES, type PostStatus } from '$lib/types';

	export interface EditorValues {
		slug: string;
		title: string;
		body: string;
		status?: PostStatus;
	}

	interface Props {
		values: EditorValues;
		/** Show the status select (posts only). */
		withStatus?: boolean;
		submitLabel?: string;
		/** Fill the slug from the title until it is edited by hand. */
		autoSlug?: boolean;
		action?: string;
	}

	let {
		values,
		withStatus = false,
		submitLabel = 'Save',
		autoSlug = false,
		action = '?/save'
	}: Props = $props();

	// Writable deriveds: they follow `values` (reset after a successful save)
	// and keep what was typed otherwise.
	let slug = $derived(values.slug);
	let title = $derived(values.title);
	let body = $derived(values.body);
	let status = $derived<PostStatus>(values.status ?? 'draft');
	let slugTouched = $derived(!autoSlug || values.slug !== '');

	let saving = $state(false);
	let showPreview = $state(true);
	let previewHtml = $state('');
	let previewError = $state('');
	let previewing = $state(false);
	// Source of the html currently shown, so the same text is not rendered twice.
	let previewSource: string | null = null;

	function slugify(text: string): string {
		return text
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '')
			.slice(0, 100);
	}

	function onTitleInput() {
		if (!slugTouched) slug = slugify(title);
	}

	$effect(() => {
		if (!showPreview) return;
		const source = body;
		if (source === previewSource) return;
		const controller = new AbortController();
		const timer = setTimeout(async () => {
			previewing = true;
			try {
				const res = await fetch('/admin/preview', {
					method: 'POST',
					headers: { 'content-type': 'application/json' },
					body: JSON.stringify({ body: source }),
					signal: controller.signal
				});
				if (!res.ok) {
					previewError =
						res.status === 401 ? 'Session expired, please log in again.' : 'Preview failed.';
					return;
				}
				previewHtml = (await res.json()).html;
				previewSource = source;
				previewError = '';
			} catch (err) {
				if ((err as Error).name !== 'AbortError') previewError = 'Preview failed.';
			} finally {
				previewing = false;
			}
		}, 400);
		return () => {
			clearTimeout(timer);
			controller.abort();
		};
	});
</script>

<form
	method="POST"
	{action}
	use:enhance={() => {
		saving = true;
		return async ({ update }) => {
			await update({ reset: false });
			saving = false;
		};
	}}
>
	<div class="fields card">
		<label class="title-field">
			Title
			<input
				name="title"
				type="text"
				required
				maxlength="200"
				bind:value={title}
				oninput={onTitleInput}
			/>
		</label>
		<label>
			Slug
			<input
				name="slug"
				type="text"
				required
				maxlength="100"
				pattern="[a-z0-9]+(-[a-z0-9]+)*"
				title="Lowercase letters, digits and single dashes"
				bind:value={slug}
				oninput={() => (slugTouched = true)}
			/>
		</label>
		{#if withStatus}
			<label>
				Status
				<select name="status" bind:value={status}>
					{#each POST_STATUSES as option (option)}
						<option value={option}>{option}</option>
					{/each}
				</select>
			</label>
		{/if}
	</div>

	<div class={showPreview ? 'panes split' : 'panes'}>
		<label class="pane">
			<span class="pane-title">Markdown</span>
			<textarea name="body" rows="24" spellcheck="true" bind:value={body}></textarea>
		</label>
		{#if showPreview}
			<!-- No aria-busy here: Pico turns it into a spinner plus white-space: nowrap
			     on the whole block, which makes the preview jump on every update. -->
			<section class="pane" aria-label="Preview">
				<span class="pane-title">
					Preview
					<span class="pane-status" class:visible={previewing} role="status">
						{previewing ? 'updating' : ''}
					</span>
				</span>
				<div class="preview">
					{#if previewError}
						<p class="preview-error">{previewError}</p>
					{/if}
					<Markdown html={previewHtml} />
				</div>
			</section>
		{/if}
	</div>

	<div class="savebar">
		<label class="preview-toggle">
			<input type="checkbox" role="switch" bind:checked={showPreview} />
			Live preview
		</label>
		<button type="submit" aria-busy={saving} disabled={saving}>{submitLabel}</button>
	</div>
</form>

<style>
	.fields {
		display: grid;
		grid-template-columns: minmax(0, 2fr) minmax(0, 1.2fr);
		gap: 0 1rem;
		margin-bottom: 1.25rem;
		padding: 1.25rem 1.25rem 0.25rem;
		border-radius: var(--radius);
	}

	.fields:has(select) {
		grid-template-columns: minmax(0, 2fr) minmax(0, 1.2fr) minmax(0, 1fr);
	}

	@media (max-width: 720px) {
		.fields,
		.fields:has(select) {
			grid-template-columns: minmax(0, 1fr);
		}
	}

	.panes {
		display: grid;
		gap: 1.25rem;
	}

	@media (min-width: 1100px) {
		.split {
			grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
		}
	}

	.pane {
		display: flex;
		flex-direction: column;
		margin: 0;
		min-width: 0;
	}

	.pane-title {
		margin-bottom: 0.4rem;
		color: var(--muted);
		font-size: 0.75rem;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
	}

	textarea {
		flex: 1;
		min-height: 65vh;
		margin: 0;
		padding: 1rem 1.1rem;
		border-radius: var(--radius);
		font-family: var(--font-mono);
		font-size: 0.875rem;
		line-height: 1.7;
		resize: vertical;
	}

	.pane-status {
		margin-left: 0.5rem;
		color: var(--accent);
		font-weight: 500;
		letter-spacing: 0;
		text-transform: none;
		opacity: 0;
		transition: opacity 0.2s var(--ease);
	}

	.pane-status.visible {
		opacity: 1;
	}

	.preview {
		flex: 1;
		min-height: 65vh;
		max-height: 80vh;
		padding: 1.25rem 1.5rem;
		box-sizing: border-box;
		overflow: auto;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
	}

	.preview-error {
		color: var(--danger);
		font-size: 0.875rem;
	}

	.savebar {
		position: sticky;
		bottom: 0;
		z-index: 5;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		margin-top: 1.25rem;
		padding: 0.85rem 0;
		border-top: 1px solid var(--border);
		background: var(--bg);
		box-shadow: 0 -8px 16px -12px rgba(0, 0, 0, 0.25);
	}

	.preview-toggle {
		margin: 0;
		color: var(--muted);
		font-size: 0.9375rem;
	}

	.savebar button {
		width: auto;
		min-width: 8rem;
		margin: 0;
	}
</style>
