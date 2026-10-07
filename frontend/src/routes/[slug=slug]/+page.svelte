<script lang="ts">
	import Markdown from '$lib/components/Markdown.svelte';
	import { SITE_NAME } from '$lib/nav';

	let { data } = $props();
</script>

<svelte:head>
	<title>{data.page.title} - {SITE_NAME}</title>
	{#if data.description}
		<meta name="description" content={data.description} />
	{/if}
</svelte:head>

<article>
	<h1>{data.page.title}</h1>
	<Markdown html={data.html} />
</article>
