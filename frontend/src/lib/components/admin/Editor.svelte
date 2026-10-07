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
	<div class="grid">
		<label>
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

	<label class="preview-toggle">
		<input type="checkbox" role="switch" bind:checked={showPreview} />
		Live preview
	</label>

	<div class={showPreview ? 'panes split' : 'panes'}>
		<label>
			Body (markdown)
			<textarea name="body" rows="24" spellcheck="true" bind:value={body}></textarea>
		</label>
		{#if showPreview}
			<section class="preview" aria-label="Preview" aria-busy={previewing}>
				{#if previewError}
					<p><small>{previewError}</small></p>
				{/if}
				<Markdown html={previewHtml} />
			</section>
		{/if}
	</div>

	<button type="submit" aria-busy={saving} disabled={saving}>{submitLabel}</button>
</form>

<style>
	.panes {
		display: grid;
		gap: 1rem;
	}

	@media (min-width: 1024px) {
		.split {
			grid-template-columns: 1fr 1fr;
		}
	}

	textarea {
		font-family: var(--pico-font-family-monospace);
		font-size: 0.875rem;
		min-height: 24rem;
	}

	.preview {
		border: 1px solid var(--pico-muted-border-color);
		border-radius: var(--pico-border-radius);
		padding: 1rem;
		margin-bottom: var(--pico-spacing);
		max-height: 80vh;
		overflow: auto;
	}

	.preview-toggle {
		margin-bottom: 1rem;
	}
</style>
