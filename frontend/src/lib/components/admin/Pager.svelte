<script lang="ts">
	import { page as current } from '$app/state';

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
	<nav aria-label="Pagination">
		<ul>
			{#if page > 1}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- only the query changes -->
				<li><a href={hrefFor(page - 1)}>Previous</a></li>
			{/if}
		</ul>
		<ul>
			<li><small>Page {page} of {pages}</small></li>
		</ul>
		<ul>
			{#if page < pages}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- only the query changes -->
				<li><a href={hrefFor(page + 1)}>Next</a></li>
			{/if}
		</ul>
	</nav>
{/if}
