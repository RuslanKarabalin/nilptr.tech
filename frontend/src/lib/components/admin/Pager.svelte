<script lang="ts">
	import { page as current } from '$app/state';
	import Icon from '../Icon.svelte';

	let { page, pages }: { page: number; pages: number } = $props();

	function hrefFor(n: number): string {
		const entries = [...current.url.searchParams].filter(([key]) => key !== 'page');
		if (n > 1) entries.push(['page', String(n)]);
		const params = new URLSearchParams(entries);
		const query = params.toString();
		return query ? `?${query}` : current.url.pathname;
	}
</script>

{#if pages > 1}
	<nav class="pager" aria-label="Pagination">
		{#if page > 1}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- only the query changes -->
			<a href={hrefFor(page - 1)}><Icon name="arrow-left" /> Previous</a>
		{:else}
			<span></span>
		{/if}
		<span class="info">Page {page} of {pages}</span>
		{#if page < pages}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- only the query changes -->
			<a href={hrefFor(page + 1)}>Next <Icon name="arrow-right" /></a>
		{:else}
			<span></span>
		{/if}
	</nav>
{/if}

<style>
	.pager {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		margin-top: 1.5rem;
	}

	a {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		padding: 0.45rem 0.9rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
		color: var(--text);
		font-size: 0.875rem;
		font-weight: 500;
		text-decoration: none;
	}

	a:hover {
		border-color: var(--accent);
		color: var(--accent);
	}

	.info {
		color: var(--muted);
		font-size: 0.875rem;
	}
</style>
