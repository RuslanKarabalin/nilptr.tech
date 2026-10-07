<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { SITE_NAME } from '$lib/nav';

	let title = $derived(page.status === 404 ? 'Page not found' : 'Something went wrong');
</script>

<svelte:head>
	<title>{page.status} {title} - {SITE_NAME}</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<section class="container error-page">
	<h1>{page.status}</h1>
	<p>{title}</p>
	{#if page.status !== 404 && page.error?.message}
		<p><small>{page.error.message}</small></p>
	{/if}
	<p><a href={resolve('/')}>Go to the home page</a></p>
</section>

<style>
	.error-page {
		padding-block: 2rem;
	}
</style>
