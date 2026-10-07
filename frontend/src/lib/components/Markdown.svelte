<script lang="ts">
	import 'katex/dist/katex.min.css';

	/** HTML produced and sanitized by $lib/server/markdown. */
	let { html }: { html: string } = $props();
</script>

<div class="markdown">
	<!-- eslint-disable-next-line svelte/no-at-html-tags -- sanitized on the server -->
	{@html html}
</div>

<style>
	.markdown :global(img),
	.markdown :global(video) {
		display: block;
		max-width: 100%;
		height: auto;
		margin-block: 1rem;
	}

	.markdown :global(.katex-display) {
		display: block;
		overflow-x: auto;
		overflow-y: hidden;
		padding-block: 0.25rem;
	}

	.markdown :global(.file-card) {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 0.5rem;
		margin-block: 1rem;
		padding: 0.75rem 1rem;
		border: 1px solid var(--pico-muted-border-color);
		border-radius: var(--pico-border-radius);
	}

	.markdown :global(.file-card-size) {
		color: var(--pico-muted-color);
		font-size: 0.875em;
	}

	.markdown :global(.file-card-missing) {
		color: var(--pico-muted-color);
	}
</style>
