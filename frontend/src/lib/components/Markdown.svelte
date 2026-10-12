<script lang="ts">
	import 'katex/dist/katex.min.css';
	import '$lib/styles/prose.css';

	/** HTML produced and sanitized by $lib/server/markdown. */
	let { html }: { html: string } = $props();
</script>

<div class="prose">
	<!-- eslint-disable-next-line svelte/no-at-html-tags -- sanitized on the server -->
	{@html html}
</div>
