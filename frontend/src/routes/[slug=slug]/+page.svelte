<script lang="ts">
	import Markdown from '$lib/components/Markdown.svelte';
	import { formatDateLong } from '$lib/format';
	import { SITE_NAME } from '$lib/nav';

	let { data } = $props();
</script>

<svelte:head>
	<title>{data.page.title} - {SITE_NAME}</title>
	{#if data.description}
		<meta name="description" content={data.description} />
	{/if}
</svelte:head>

<div class="wrap wrap-prose">
	<article class="bare">
		<header class="page-header">
			<h1>{data.page.title}</h1>
			{#if data.page.updated_at}
				<p class="eyebrow">
					Updated <time datetime={data.page.updated_at}>{formatDateLong(data.page.updated_at)}</time
					>
				</p>
			{/if}
		</header>
		<div class="page-body card">
			<Markdown html={data.html} />
		</div>
	</article>
</div>

<style>
	.page-header {
		margin-bottom: 2rem;
	}

	.page-header h1 {
		margin: 0 0 0.5rem;
		font-size: clamp(1.9rem, 4.5vw, 2.75rem);
		line-height: 1.15;
		letter-spacing: -0.035em;
	}

	.page-header .eyebrow {
		margin: 0;
	}

	.page-body {
		padding: clamp(1.25rem, 4vw, 2.75rem);
		border-radius: var(--radius-lg);
	}
</style>
